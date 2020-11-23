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

// Directory used by Maven for an invocation of the kamel local command.
// By default a temporary folder will be used.
var mavenWorkingDirectory string = ""

var acceptedDependencyTypes = []string{"bom", "camel", "camel-k", "camel-quarkus", "mvn", "github"}

var additionalDependencyUsageMessage = `Additional top-level dependencies are specified with the format:
<type>:<dependency-name>
where <type> is one of {` + strings.Join(acceptedDependencyTypes, "|") + `}.`

const defaultDependenciesDirectoryName = "dependencies"

func getDependencies(args []string, additionalDependencies []string, allDependencies bool) ([]string, error) {
	// Fetch existing catalog or create new one if one does not already exist.
	catalog, err := createCamelCatalog()

	// Get top-level dependencies.
	dependencies, err := getTopLevelDependencies(catalog, args)
	if err != nil {
		return nil, err
	}

	// Add additional user-provided dependencies.
	if additionalDependencies != nil {
		for _, additionalDependency := range additionalDependencies {
			dependencies = append(dependencies, additionalDependency)
		}
	}

	// Compute transitive dependencies.
	if allDependencies {
		dependencies, err = getTransitiveDependencies(catalog, dependencies)
		if err != nil {
			return nil, err
		}
	}

	return dependencies, nil
}

func getTopLevelDependencies(catalog *camel.RuntimeCatalog, args []string) ([]string, error) {
	// List of top-level dependencies.
	dependencies := strset.New()

	// Invoke the dependency inspector code for each source file.
	for _, source := range args {
		data, _, err := loadContent(source, false, false)
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

		// Extract list of top-level dependencies.
		dependencies.Merge(trait.AddSourceDependencies(sourceSpec, catalog))
	}

	return dependencies.List(), nil
}

func getTransitiveDependencies(
	catalog *camel.RuntimeCatalog,
	dependencies []string) ([]string, error) {

	mvn := v1.MavenSpec{
		LocalRepository: "",
	}

	// Create Maven project.
	project := runtime.GenerateQuarkusProjectCommon(
		catalog.CamelCatalogSpec.Runtime.Metadata["camel-quarkus.version"],
		defaults.DefaultRuntimeVersion, catalog.CamelCatalogSpec.Runtime.Metadata["quarkus.version"])

	// Inject dependencies into Maven project.
	err := camel.ManageIntegrationDependencies(&project, dependencies, catalog)
	if err != nil {
		return nil, err
	}

	// Maven local context to be used for generating the transitive dependencies.
	mc := maven.NewContext(mavenWorkingDirectory, project)
	mc.LocalRepository = mvn.LocalRepository
	mc.Timeout = mvn.GetTimeout().Duration

	// Make maven command less verbose.
	mc.AdditionalArguments = append(mc.AdditionalArguments, "-q")

	err = runtime.BuildQuarkusRunnerCommon(mc)
	if err != nil {
		return nil, err
	}

	// Compute dependencies.
	content, err := runtime.ComputeQuarkusDependenciesCommon(mc, catalog.Runtime.Version)
	if err != nil {
		return nil, err
	}

	// Compose artifacts list.
	artifacts := []v1.Artifact{}
	artifacts, err = runtime.ProcessQuarkusTransitiveDependencies(mc, content)
	if err != nil {
		return nil, err
	}

	// Dump dependencies in the dependencies directory and construct the list of dependencies.
	transitiveDependencies := []string{}
	for _, entry := range artifacts {
		transitiveDependencies = append(transitiveDependencies, entry.Location)
	}

	return transitiveDependencies, nil
}

func generateCatalog() (*camel.RuntimeCatalog, error) {
	// A Camel catalog is requiref for this operatio.
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
	// Attempt to reuse existing Camel catalog if one is present.
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}

	// Generate catalog if one was not found.
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
		// Print output in text form.
		for _, dep := range dependencies {
			fmt.Printf("%v\n", dep)
		}
	}

	return nil
}

func printDependencies(format string, dependecies []string) error {
	switch format {
	case "yaml":
		data, err := util.DependenciesToYAML(dependecies)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	case "json":
		data, err := util.DependenciesToJSON(dependecies)
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

	// Report any error.
	if err != nil {
		return err
	}

	// Signal file not found.
	if !fileExists {
		return errors.New("File " + file + " file does not exist")
	}

	return nil
}

func validateFiles(args []string) error {
	// Ensure source files exist.
	for _, arg := range args {
		err := validateFile(arg)
		if err != nil {
			return nil
		}
	}

	return nil
}

func validateAdditionalDependencies(additionalDependencies []string) error {
	// Validate list of additional dependencies i.e. make sure that each dependency has
	// a valid type.
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

func getPropertiesDir() string {
	// Directory is created under the maven directory which is removed.
	return path.Join(mavenWorkingDirectory, "properties")
}

func createPropertiesDirectory() error {
	// Check directory exists.
	directoryExists, err := util.DirectoryExists(getPropertiesDir())
	if err != nil {
		return err
	}

	if !directoryExists {
		err := os.MkdirAll(getPropertiesDir(), 0777)
		if err != nil {
			return err
		}
	}
	return nil
}

func updateIntegrationProperties(properties []string, propertyFiles []string) ([]string, error) {
	// Create properties directory under Maven working directory. This ensures that
	// property files of different integrations do not clash.
	err := createPropertiesDirectory()
	if err != nil {
		return nil, err
	}

	// Relocate properties files to this integration's property directory.
	relocatedPropertyFiles := []string{}
	for _, propertyFile := range propertyFiles {
		relocatedPropertyFile := path.Join(getPropertiesDir(), path.Base(propertyFile))
		util.CopyFile(propertyFile, relocatedPropertyFile)
		relocatedPropertyFiles = append(relocatedPropertyFiles, relocatedPropertyFile)
	}

	// Output list of properties to property file if any CLI properties were given.
	if len(properties) > 0 {
		propertyFilePath := path.Join(getPropertiesDir(), "CLI.properties")
		err = ioutil.WriteFile(propertyFilePath, []byte(strings.Join(properties, "\n")), 0777)
		if err != nil {
			return nil, err
		}
		relocatedPropertyFiles = append(relocatedPropertyFiles, propertyFilePath)
	}

	// Return relocated PropertyFiles.
	return relocatedPropertyFiles, nil
}

func createMavenWorkingDirectory() error {
	// Create local Maven context.
	temporaryDirectory, err := ioutil.TempDir(os.TempDir(), "maven-")
	if err != nil {
		return err
	}

	// Set the Maven directory to the default value.
	mavenWorkingDirectory = temporaryDirectory

	return nil
}

func deleteMavenWorkingDirectory() error {
	// Remove directory used for computing the dependencies.
	defer os.RemoveAll(mavenWorkingDirectory)

	return nil
}
