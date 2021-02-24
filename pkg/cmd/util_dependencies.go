/*
Licensed to the Apache Software Foundation (ASF) under one or more
contributor license agreements.  See the NOTICE file distributed with
this work for additional information regarding copyright ownership.
The ASF licenses this file to You under the Apache License, Version 2.0
(the "License"); you may not use this file except in compliance with
the License.  You may obtain a copy of the License at

   http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package cmd

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder/runtime"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"
)

var acceptedDependencyTypes = []string{"bom", "camel", "camel-k", "camel-quarkus", "mvn", "github"}

var additionalDependencyUsageMessage = `Additional top-level dependencies are specified with the format:
<type>:<dependency-name>
where <type> is one of {` + strings.Join(acceptedDependencyTypes, "|") + `}.`

func getDependencies(args []string, additionalDependencies []string, repositories []string, allDependencies bool) ([]string, error) {
	// Fetch existing catalog or create new one if one does not already exist
	catalog, err := createCamelCatalog()

	// Get top-level dependencies
	dependencies, err := getTopLevelDependencies(catalog, args)
	if err != nil {
		return nil, err
	}

	// Add additional user-provided dependencies
	if additionalDependencies != nil {
		for _, additionalDependency := range additionalDependencies {
			dependencies = append(dependencies, additionalDependency)
		}
	}

	// Compute transitive dependencies
	if allDependencies {
		// Add runtime dependency since this dependency is always required for running
		// an integration. Only add this dependency if it has not been added already.
		for _, runtimeDep := range catalog.Runtime.Dependencies {
			util.StringSliceUniqueAdd(&dependencies, runtimeDep.GetDependencyID())
		}

		dependencies, err = getTransitiveDependencies(catalog, dependencies, repositories)
		if err != nil {
			return nil, err
		}
	}
	return dependencies, nil
}

func getTopLevelDependencies(catalog *camel.RuntimeCatalog, args []string) ([]string, error) {
	// List of top-level dependencies
	dependencies := strset.New()

	// Invoke the dependency inspector code for each source file
	for _, source := range args {
		data, _, _, err := loadTextContent(source, false)
		if err != nil {
			return []string{}, err
		}

		sourceSpec := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(source),
				Content:     data,
				Compression: false,
			},
		}

		// Extract list of top-level dependencies
		dependencies.Merge(trait.AddSourceDependencies(sourceSpec, catalog))
	}

	return dependencies.List(), nil
}

func getTransitiveDependencies(
	catalog *camel.RuntimeCatalog,
	dependencies []string, repositories []string) ([]string, error) {

	mvn := v1.MavenSpec{
		LocalRepository: "",
	}

	// Create Maven project
	project := runtime.GenerateQuarkusProjectCommon(
		catalog.CamelCatalogSpec.Runtime.Metadata["camel-quarkus.version"],
		defaults.DefaultRuntimeVersion, catalog.CamelCatalogSpec.Runtime.Metadata["quarkus.version"])

	// Inject dependencies into Maven project
	err := camel.ManageIntegrationDependencies(&project, dependencies, catalog)
	if err != nil {
		return nil, err
	}

	// Maven local context to be used for generating the transitive dependencies
	mc := maven.NewContext(util.MavenWorkingDirectory, project)
	mc.LocalRepository = mvn.LocalRepository
	mc.Timeout = mvn.GetTimeout().Duration

	if len(repositories) > 0 {
		var repoList []maven.Repository
		for i, repo := range repositories {
			repository := maven.NewRepository(repo)
			if repository.ID == "" {
				repository.ID = fmt.Sprintf("repository-%03d", i)
			}
			repoList = append(repoList, repository)
		}

		settings := maven.NewDefaultSettings(repoList)
		settingsData, err := util.EncodeXML(settings)
		if err != nil {
			return nil, err
		}
		mc.SettingsContent = settingsData
	}

	// Make maven command less verbose
	mc.AdditionalArguments = append(mc.AdditionalArguments, "-q")

	err = runtime.BuildQuarkusRunnerCommon(mc)
	if err != nil {
		return nil, err
	}

	// Compose artifacts list
	artifacts := []v1.Artifact{}
	artifacts, err = runtime.ProcessQuarkusTransitiveDependencies(mc)
	if err != nil {
		return nil, err
	}

	// Dump dependencies in the dependencies directory and construct the list of dependencies
	transitiveDependencies := []string{}
	for _, entry := range artifacts {
		transitiveDependencies = append(transitiveDependencies, entry.Location)
	}
	return transitiveDependencies, nil
}

func getRegularFilesInDir(directory string) ([]string, error) {
	dirFiles := []string{}
	files, err := ioutil.ReadDir(directory)
	for _, file := range files {
		fileName := file.Name()

		// Do not include hidden files or sub-directories.
		if !file.IsDir() && !strings.HasPrefix(fileName, ".") {
			dirFiles = append(dirFiles, path.Join(directory, fileName))
		}
	}

	if err != nil {
		return nil, err
	}

	return dirFiles, nil
}

func getLocalBuildDependencies(integrationDirectory string) ([]string, error) {
	locallyBuiltDependencies, err := getRegularFilesInDir(getCustomDependenciesDir(integrationDirectory))
	if err != nil {
		return nil, err
	}
	return locallyBuiltDependencies, nil
}

func getLocalBuildProperties(integrationDirectory string) ([]string, error) {
	locallyBuiltProperties, err := getRegularFilesInDir(getCustomPropertiesDir(integrationDirectory))
	if err != nil {
		return nil, err
	}
	return locallyBuiltProperties, nil
}

func getLocalBuildRoutes(integrationDirectory string) ([]string, error) {
	locallyBuiltRoutes, err := getRegularFilesInDir(getCustomRoutesDir(integrationDirectory))
	if err != nil {
		return nil, err
	}
	return locallyBuiltRoutes, nil
}

func generateCatalog() (*camel.RuntimeCatalog, error) {
	// A Camel catalog is required for this operation
	settings := ""
	mvn := v1.MavenSpec{
		LocalRepository: "",
	}
	runtime := v1.RuntimeSpec{
		Version:  defaults.DefaultRuntimeVersion,
		Provider: v1.RuntimeProviderQuarkus,
	}
	providerDependencies := []maven.Dependency{}
	catalog, err := camel.GenerateCatalogCommon(settings, mvn, runtime, providerDependencies)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func createCamelCatalog() (*camel.RuntimeCatalog, error) {
	// Attempt to reuse existing Camel catalog if one is present
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}

	// Generate catalog if one was not found
	if catalog == nil {
		catalog, err = generateCatalog()
		if err != nil {
			return nil, err
		}
	}

	return catalog, nil
}

func outputDependencies(dependencies []string, format string) error {
	if format != "" {
		err := printDependencies(format, dependencies)
		if err != nil {
			return err
		}
	} else {
		// Print output in text form
		for _, dep := range dependencies {
			fmt.Printf("%v\n", dep)
		}
	}

	return nil
}

func printDependencies(format string, dependencies []string) error {
	switch format {
	case "yaml":
		data, err := util.DependenciesToYAML(dependencies)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	case "json":
		data, err := util.DependenciesToJSON(dependencies)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	default:
		return errors.New("unknown output format: " + format)
	}
	return nil
}

func validateFile(file string) error {
	fileExists, err := util.FileExists(file)

	if err != nil {
		return err
	}

	if !fileExists {
		return errors.New("File " + file + " file does not exist")
	}

	return nil
}

func validateFiles(args []string) error {
	// Ensure source files exist
	for _, arg := range args {
		err := validateFile(arg)
		if err != nil {
			return nil
		}
	}

	return nil
}

func validateAdditionalDependencies(additionalDependencies []string) error {
	// Validate list of additional dependencies i.e. make sure that each dependency has a valid type
	if additionalDependencies != nil {
		for _, additionalDependency := range additionalDependencies {
			isValid := validateDependency(additionalDependency)
			if !isValid {
				return errors.New("Unexpected type for user-provided dependency: " + additionalDependency + ". " + additionalDependencyUsageMessage)
			}
		}
	}

	return nil
}

func validateDependency(additionalDependency string) bool {
	dependencyComponents := strings.Split(additionalDependency, ":")

	TypeIsValid := false
	for _, dependencyType := range acceptedDependencyTypes {
		if dependencyType == dependencyComponents[0] {
			TypeIsValid = true
		}
	}

	return TypeIsValid
}

func validateIntegrationFiles(args []string) error {
	// If no source files have been provided there is nothing to inspect.
	if len(args) == 0 {
		return errors.New("no integration files have been provided")
	}

	// Validate integration files.
	err := validateFiles(args)
	if err != nil {
		return nil
	}

	return nil
}

func validatePropertyFiles(propertyFiles []string) error {
	for _, fileName := range propertyFiles {
		if !strings.HasSuffix(fileName, ".properties") {
			return fmt.Errorf("supported property files must have a .properties extension: %s", fileName)
		}

		if file, err := os.Stat(fileName); err != nil {
			return errors.Wrapf(err, "unable to access property file %s", fileName)
		} else if file.IsDir() {
			return fmt.Errorf("property file %s is a directory", fileName)
		}
	}

	return nil
}

func updateIntegrationProperties(properties []string, propertyFiles []string, hasIntegrationDir bool) ([]string, error) {
	// Create properties directory under Maven working directory.
	// This ensures that property files of different integrations do not clash.
	err := util.CreateLocalPropertiesDirectory()
	if err != nil {
		return nil, err
	}

	// Relocate properties files to this integration's property directory.
	relocatedPropertyFiles := []string{}
	for _, propertyFile := range propertyFiles {
		relocatedPropertyFile := path.Join(util.GetLocalPropertiesDir(), path.Base(propertyFile))
		util.CopyFile(propertyFile, relocatedPropertyFile)
		relocatedPropertyFiles = append(relocatedPropertyFiles, relocatedPropertyFile)
	}

	if !hasIntegrationDir {
		// Output list of properties to property file if any CLI properties were given.
		if len(properties) > 0 {
			propertyFilePath := path.Join(util.GetLocalPropertiesDir(), "CLI.properties")
			err = ioutil.WriteFile(propertyFilePath, []byte(strings.Join(properties, "\n")), 0777)
			if err != nil {
				return nil, err
			}
			relocatedPropertyFiles = append(relocatedPropertyFiles, propertyFilePath)
		}
	}

	return relocatedPropertyFiles, nil
}

func updateIntegrationDependencies(dependencies []string) error {
	// Create dependencies directory under Maven working directory.
	// This ensures that dependencies will be removed after they are not needed.
	err := util.CreateLocalDependenciesDirectory()
	if err != nil {
		return err
	}

	// Relocate dependencies files to this integration's dependencies directory
	for _, dependency := range dependencies {
		var targetPath string
		basePath := util.SubstringFrom(dependency, util.QuarkusDependenciesBaseDirectory)
		if basePath != "" {
			targetPath = path.Join(util.GetLocalDependenciesDir(), basePath)
		} else {
			targetPath = path.Join(util.GetLocalDependenciesDir(), path.Base(dependency))
		}
		util.CopyFile(dependency, targetPath)
	}

	return nil
}

func updateIntegrationRoutes(routes []string) error {
	err := util.CreateLocalRoutesDirectory()
	if err != nil {
		return err
	}

	for _, route := range routes {
		util.CopyFile(route, path.Join(util.GetLocalRoutesDir(), path.Base(route)))
	}

	return nil
}

func createMavenWorkingDirectory() error {
	// Create local Maven context
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "maven-")
	if err != nil {
		return err
	}

	// Set the Maven directory to the default value
	util.MavenWorkingDirectory = temporaryDirectory

	return nil
}

func deleteMavenWorkingDirectory() error {
	// Remove directory used for computing the dependencies
	defer os.RemoveAll(util.MavenWorkingDirectory)

	return nil
}

func getCustomDependenciesDir(integrationDirectory string) string {
	return path.Join(integrationDirectory, "dependencies")
}

func getCustomPropertiesDir(integrationDirectory string) string {
	return path.Join(integrationDirectory, "properties")
}

func getCustomRoutesDir(integrationDirectory string) string {
	return path.Join(integrationDirectory, "routes")
}
