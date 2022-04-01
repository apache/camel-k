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
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"
	"github.com/spf13/cobra"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

var acceptedDependencyTypes = []string{"bom", "camel", "camel-k", "camel-quarkus", "mvn", "github"}

var additionalDependencyUsageMessage = `Additional top-level dependencies are specified with the format:
<type>:<dependency-name>
where <type> is one of {` + strings.Join(acceptedDependencyTypes, "|") + `}.`

func getDependencies(ctx context.Context, args []string, additionalDependencies []string, repositories []string, allDependencies bool) ([]string, error) {
	// Fetch existing catalog or create new one if one does not already exist
	catalog, err := createCamelCatalog(ctx)
	if err != nil {
		return nil, err
	}

	// Get top-level dependencies
	dependencies, err := getTopLevelDependencies(ctx, catalog, args)
	if err != nil {
		return nil, err
	}

	// Add additional user-provided dependencies
	dependencies = append(dependencies, additionalDependencies...)

	// Compute transitive dependencies
	if allDependencies {
		// Add runtime dependency since this dependency is always required for running
		// an integration. Only add this dependency if it has not been added already.
		for _, runtimeDep := range catalog.Runtime.Dependencies {
			util.StringSliceUniqueAdd(&dependencies, runtimeDep.GetDependencyID())
		}

		dependencies, err = getTransitiveDependencies(ctx, catalog, dependencies, repositories)
		if err != nil {
			return nil, err
		}
	}
	return dependencies, nil
}

func getTopLevelDependencies(ctx context.Context, catalog *camel.RuntimeCatalog, args []string) ([]string, error) {
	// List of top-level dependencies
	dependencies := strset.New()

	// Invoke the dependency inspector code for each source file
	for _, source := range args {
		data, _, _, err := loadTextContent(ctx, source, false)
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

func getTransitiveDependencies(ctx context.Context, catalog *camel.RuntimeCatalog, dependencies []string, repositories []string) ([]string, error) {
	project := builder.GenerateQuarkusProjectCommon(
		catalog.CamelCatalogSpec.Runtime.Metadata["camel-quarkus.version"],
		defaults.DefaultRuntimeVersion,
		catalog.CamelCatalogSpec.Runtime.Metadata["quarkus.version"],
	)

	err := camel.ManageIntegrationDependencies(&project, dependencies, catalog)
	if err != nil {
		return nil, err
	}

	mc := maven.NewContext(util.MavenWorkingDirectory)
	mc.LocalRepository = ""

	if len(repositories) > 0 {
		settings, err := maven.NewSettings(maven.DefaultRepositories, maven.Repositories(repositories...))
		if err != nil {
			return nil, err
		}
		settingsData, err := util.EncodeXML(settings)
		if err != nil {
			return nil, err
		}
		mc.UserSettings = settingsData
	}

	// Make maven command less verbose
	mc.AdditionalArguments = append(mc.AdditionalArguments, "-q")

	err = builder.BuildQuarkusRunnerCommon(ctx, mc, project)
	if err != nil {
		return nil, err
	}

	// Compose artifacts list
	artifacts, err := builder.ProcessQuarkusTransitiveDependencies(mc)
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
	var dirFiles []string
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

func generateCatalog(ctx context.Context) (*camel.RuntimeCatalog, error) {
	// A Camel catalog is required for this operation
	mvn := v1.MavenSpec{
		LocalRepository: "",
	}
	runtime := v1.RuntimeSpec{
		Version:  defaults.DefaultRuntimeVersion,
		Provider: v1.RuntimeProviderQuarkus,
	}
	var providerDependencies []maven.Dependency
	var caCert []byte
	catalog, err := camel.GenerateCatalogCommon(ctx, nil, nil, caCert, mvn, runtime, providerDependencies)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func createCamelCatalog(ctx context.Context) (*camel.RuntimeCatalog, error) {
	// Attempt to reuse existing Camel catalog if one is present
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}

	// Generate catalog if one was not found
	if catalog == nil {
		catalog, err = generateCatalog(ctx)
		if err != nil {
			return nil, err
		}
	}

	return catalog, nil
}

func outputDependencies(dependencies []string, format string, cmd *cobra.Command) error {
	if format != "" {
		err := printDependencies(format, dependencies, cmd)
		if err != nil {
			return err
		}
	} else {
		// Print output in text form
		fmt.Fprintln(cmd.OutOrStdout(), "dependencies:")
		for _, dep := range dependencies {
			fmt.Fprintln(cmd.OutOrStdout(), dep)
		}
	}

	return nil
}

func printDependencies(format string, dependencies []string, cmd *cobra.Command) error {
	switch format {
	case "yaml":
		data, err := util.DependenciesToYAML(dependencies)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(data))
	case "json":
		data, err := util.DependenciesToJSON(dependencies)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(data))
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
			return err
		}
	}

	return nil
}

func validateAdditionalDependencies(additionalDependencies []string) error {
	// Validate list of additional dependencies i.e. make sure that each dependency has a valid type
	for _, additionalDependency := range additionalDependencies {
		isValid := validateDependency(additionalDependency)
		if !isValid {
			return errors.New("Unexpected type for user-provided dependency: " + additionalDependency + ". " + additionalDependencyUsageMessage)
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
	if err := validateFiles(args); err != nil {
		return err
	}

	return nil
}

func validatePropertyFiles(propertyFiles []string) error {
	for _, fileName := range propertyFiles {
		if err := validatePropertyFile(fileName); err != nil {
			return err
		}
	}

	return nil
}

func validatePropertyFile(fileName string) error {
	if !strings.HasSuffix(fileName, ".properties") {
		return fmt.Errorf("supported property files must have a .properties extension: %s", fileName)
	}

	if file, err := os.Stat(fileName); err != nil {
		return errors.Wrapf(err, "unable to access property file %s", fileName)
	} else if file.IsDir() {
		return fmt.Errorf("property file %s is a directory", fileName)
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
		_, err = util.CopyFile(propertyFile, relocatedPropertyFile)
		if err != nil {
			return nil, err
		}
		relocatedPropertyFiles = append(relocatedPropertyFiles, relocatedPropertyFile)
	}

	if !hasIntegrationDir {
		// Output list of properties to property file if any CLI properties were given.
		if len(properties) > 0 {
			propertyFilePath := path.Join(util.GetLocalPropertiesDir(), "CLI.properties")
			err = ioutil.WriteFile(propertyFilePath, []byte(strings.Join(properties, "\n")), 0o600)
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
		_, err = util.CopyFile(dependency, targetPath)
		if err != nil {
			return err
		}
	}

	return nil
}

func updateIntegrationRoutes(routes []string) error {
	err := util.CreateLocalRoutesDirectory()
	if err != nil {
		return err
	}

	for _, route := range routes {
		_, err = util.CopyFile(route, path.Join(util.GetLocalRoutesDir(), path.Base(route)))
		if err != nil {
			return err
		}
	}

	return nil
}

func updateQuarkusDirectory() error {
	err := util.CreateLocalQuarkusDirectory()
	if err != nil {
		return err
	}

	// ignore error if custom dir doesn't exist
	_ = util.CopyQuarkusAppFiles(util.CustomQuarkusDirectoryName, util.GetLocalQuarkusDir())

	return nil
}

func updateAppDirectory() error {
	err := util.CreateLocalAppDirectory()
	if err != nil {
		return err
	}

	// ignore error if custom dir doesn't exist
	_ = util.CopyAppFile(util.CustomAppDirectoryName, util.GetLocalAppDir())

	return nil
}

func updateLibDirectory() error {
	err := util.CreateLocalLibDirectory()
	if err != nil {
		return err
	}

	// ignore error if custom dir doesn't exist
	_ = util.CopyLibFiles(util.CustomLibDirectoryName, util.GetLocalLibDir())

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
	return os.RemoveAll(util.MavenWorkingDirectory)
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

func getCustomQuarkusDir(integrationDirectory string) string {
	parentDir := path.Dir(strings.TrimSuffix(integrationDirectory, "/"))
	return path.Join(parentDir, "quarkus")
}

func getCustomLibDir(integrationDirectory string) string {
	parentDir := path.Dir(strings.TrimSuffix(integrationDirectory, "/"))
	return path.Join(parentDir, "lib/main")
}

func getCustomAppDir(integrationDirectory string) string {
	parentDir := path.Dir(strings.TrimSuffix(integrationDirectory, "/"))
	return path.Join(parentDir, "app")
}

func deleteLocalIntegrationDirs(integrationDirectory string) error {
	dirs := []string{
		getCustomQuarkusDir(integrationDirectory),
		getCustomLibDir(integrationDirectory),
		getCustomAppDir(integrationDirectory),
	}

	for _, dir := range dirs {
		err := os.RemoveAll(dir)
		if err != nil {
			return err
		}
	}

	return nil
}
