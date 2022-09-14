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

package local

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"sort"
	"strings"

	"github.com/pkg/errors"
	"github.com/scylladb/go-set/strset"
	"github.com/spf13/cobra"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/cmd/source"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
)

var acceptedDependencyTypes = []string{
	"bom", "camel", "camel-k", "camel-quarkus", "mvn",
	// jitpack
	"github", "gitlab", "bitbucket", "gitee", "azure",
}

// GetDependencies resolves and gets the list of dependencies from catalog and sources.
func GetDependencies(ctx context.Context, cmd *cobra.Command, srcs, userDependencies, repositories []string,
	allDependencies bool) ([]string, error) {
	// Fetch existing catalog or create new one if one does not already exist
	catalog, err := CreateCamelCatalog(ctx)
	if err != nil {
		return nil, errors.Wrap(err, "failed to create Camel catalog")
	}

	// Validate user-provided dependencies against Camel catalog
	camel.ValidateDependencies(catalog, userDependencies, cmd)

	// Get top-level dependencies from sources
	dependencies, err := getTopLevelDependencies(ctx, catalog, srcs)
	if err != nil {
		return nil, errors.Wrap(err, "failed to resolve dependencies from source")
	}

	// Add additional user-provided dependencies
	dependencies = append(dependencies, userDependencies...)

	// Compute transitive dependencies
	if allDependencies {
		// Add runtime dependency since this dependency is always required for running
		// an integration. Only add this dependency if it has not been added already.
		for _, runtimeDep := range catalog.Runtime.Dependencies {
			util.StringSliceUniqueAdd(&dependencies, runtimeDep.GetDependencyID())
		}

		dependencies, err = getTransitiveDependencies(ctx, catalog, dependencies, repositories)
		if err != nil {
			return nil, errors.Wrap(err, "failed to compute transitive dependencies")
		}
	}

	return dependencies, nil
}

// getTopLevelDependencies gets top-level dependencies from sources.
func getTopLevelDependencies(ctx context.Context, catalog *camel.RuntimeCatalog, srcs []string) ([]string, error) {
	// List of top-level dependencies
	dependencies := strset.New()

	// Invoke the dependency inspector code for each source file
	for _, src := range srcs {
		data, _, _, err := source.LoadTextContent(ctx, src, false)
		if err != nil {
			return []string{}, err
		}

		sourceSpec := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(src),
				Content:     data,
				Compression: false,
			},
		}

		// Extract list of top-level dependencies
		srcDeps, err := trait.ExtractSourceDependencies(sourceSpec, catalog)
		if err != nil {
			return nil, err
		}
		dependencies.Merge(srcDeps)
	}

	return dependencies.List(), nil
}

func getTransitiveDependencies(ctx context.Context, catalog *camel.RuntimeCatalog, dependencies []string, repositories []string) ([]string, error) {
	project := builder.GenerateQuarkusProjectCommon(
		catalog.GetCamelQuarkusVersion(),
		defaults.DefaultRuntimeVersion,
		catalog.GetQuarkusVersion(),
	)

	if err := camel.ManageIntegrationDependencies(&project, dependencies, catalog); err != nil {
		return nil, err
	}

	mc := maven.NewContext(MavenWorkingDirectory)
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

	if err := builder.BuildQuarkusRunnerCommon(ctx, mc, project); err != nil {
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

func getRegularFilesInDir(directory string, dirnameInPath bool) ([]string, error) {
	var dirFiles []string
	files, err := ioutil.ReadDir(directory)
	if err != nil {
		return nil, err
	}

	for _, file := range files {
		fileName := file.Name()

		// Do not include hidden files or sub-directories.
		if !file.IsDir() && !strings.HasPrefix(fileName, ".") {
			if dirnameInPath {
				dirFiles = append(dirFiles, path.Join(directory, fileName))
			} else {
				dirFiles = append(dirFiles, fileName)
			}
		}
	}

	return dirFiles, nil
}

func GetBuildDependencies(integrationDirectory string) ([]string, error) {
	locallyBuiltDependencies, err := getRegularFilesInDir(GetCustomDependenciesDir(integrationDirectory), true)
	if err != nil {
		return nil, err
	}
	return locallyBuiltDependencies, nil
}

func GetBuildProperties(integrationDirectory string) ([]string, error) {
	locallyBuiltProperties, err := getRegularFilesInDir(GetCustomPropertiesDir(integrationDirectory), true)
	if err != nil {
		return nil, err
	}
	return locallyBuiltProperties, nil
}

func GetBuildRoutes(integrationDirectory string) ([]string, error) {
	locallyBuiltRoutes, err := getRegularFilesInDir(GetCustomRoutesDir(integrationDirectory), true)
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
	var caCert [][]byte
	catalog, err := camel.GenerateCatalogCommon(ctx, nil, nil, caCert, mvn, runtime, providerDependencies)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func CreateCamelCatalog(ctx context.Context) (*camel.RuntimeCatalog, error) {
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

func OutputDependencies(dependencies []string, format string, cmd *cobra.Command) error {
	sort.Strings(dependencies)
	if format != "" {
		if err := printDependencies(format, dependencies, cmd); err != nil {
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
		data, err := dependenciesToYAML(dependencies)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(data))
	case "json":
		data, err := dependenciesToJSON(dependencies)
		if err != nil {
			return err
		}
		fmt.Fprint(cmd.OutOrStdout(), string(data))
	default:
		return errors.Errorf("unknown output format: %s", format)
	}
	return nil
}

func dependenciesToJSON(list []string) ([]byte, error) {
	jsondata := map[string]interface{}{}
	jsondata["dependencies"] = list
	return json.Marshal(jsondata)
}

func dependenciesToYAML(list []string) ([]byte, error) {
	data, err := dependenciesToJSON(list)
	if err != nil {
		return nil, err
	}

	return util.JSONToYAML(data)
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

// ValidateFiles ensures existence of given files.
func ValidateFiles(args []string) error {
	// Ensure source files exist
	for _, arg := range args {
		err := validateFile(arg)
		if err != nil {
			return err
		}
	}

	return nil
}

// ValidateDependencies validates list of additional dependencies i.e. makes sure
// that each dependency has a valid type.
func ValidateDependencies(dependencies []string) error {
	for _, dependency := range dependencies {
		depType := strings.Split(dependency, ":")[0]
		if !util.StringSliceExists(acceptedDependencyTypes, depType) {
			return fmt.Errorf("dependency is not valid: %s", dependency)
		}
	}

	return nil
}

func ValidatePropertyFiles(propertyFiles []string) error {
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

func UpdateIntegrationProperties(properties []string, propertyFiles []string, hasIntegrationDir bool) ([]string, error) {
	// Create properties directory under Maven working directory.
	// This ensures that property files of different integrations do not clash.
	if err := CreateLocalPropertiesDirectory(); err != nil {
		return nil, err
	}

	// Relocate properties files to this integration's property directory.
	relocatedPropertyFiles := []string{}
	dir := GetLocalPropertiesDir()
	for _, propertyFile := range propertyFiles {
		relocatedPropertyFile := path.Join(dir, path.Base(propertyFile))
		if _, err := util.CopyFile(propertyFile, relocatedPropertyFile); err != nil {
			return nil, err
		}
		relocatedPropertyFiles = append(relocatedPropertyFiles, relocatedPropertyFile)
	}

	if !hasIntegrationDir {
		// Output list of properties to property file if any CLI properties were given.
		if len(properties) > 0 {
			propertyFilePath := path.Join(dir, "CLI.properties")
			if err := ioutil.WriteFile(propertyFilePath, []byte(strings.Join(properties, "\n")), 0o600); err != nil {
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
	if err := createLocalDependenciesDirectory(); err != nil {
		return err
	}

	// Relocate dependencies files to this integration's dependencies directory
	dir := getLocalDependenciesDir()
	for _, dependency := range dependencies {
		var targetPath string
		basePath := util.SubstringFrom(dependency, util.QuarkusDependenciesBaseDirectory)
		if basePath != "" {
			targetPath = path.Join(dir, basePath)
		} else {
			targetPath = path.Join(dir, path.Base(dependency))
		}
		if _, err := util.CopyFile(dependency, targetPath); err != nil {
			return err
		}
	}

	return nil
}

func updateIntegrationRoutes(routes []string) error {
	if err := createLocalRoutesDirectory(); err != nil {
		return err
	}

	dir := getLocalRoutesDir()
	for _, route := range routes {
		if _, err := util.CopyFile(route, path.Join(dir, path.Base(route))); err != nil {
			return err
		}
	}

	return nil
}

func updateQuarkusDirectory() error {
	if err := createLocalQuarkusDirectory(); err != nil {
		return err
	}

	// ignore error if custom dir doesn't exist
	_ = CopyQuarkusAppFiles(util.CustomQuarkusDirectoryName, getLocalQuarkusDir())

	return nil
}

func updateAppDirectory() error {
	if err := createLocalAppDirectory(); err != nil {
		return err
	}

	// ignore error if custom dir doesn't exist
	_ = CopyAppFile(util.CustomAppDirectoryName, getLocalAppDir())

	return nil
}

func updateLibDirectory() error {
	if err := createLocalLibDirectory(); err != nil {
		return err
	}

	// ignore error if custom dir doesn't exist
	_ = CopyLibFiles(util.CustomLibDirectoryName, getLocalLibDir())

	return nil
}

func CopyIntegrationFilesToDirectory(files []string, directory string) ([]string, error) {
	// Create directory if one does not already exist
	if err := util.CreateDirectory(directory); err != nil {
		return nil, err
	}

	// Copy files to new location. Also create the list with relocated files.
	relocatedFilesList := []string{}
	for _, filePath := range files {
		newFilePath := path.Join(directory, path.Base(filePath))
		if _, err := util.CopyFile(filePath, newFilePath); err != nil {
			return relocatedFilesList, err
		}
		relocatedFilesList = append(relocatedFilesList, newFilePath)
	}

	return relocatedFilesList, nil
}

func CopyQuarkusAppFiles(dependenciesDir string, quarkusDir string) error {
	// Create directory if one does not already exist
	if err := util.CreateDirectory(quarkusDir); err != nil {
		return err
	}

	// Transfer all files with a .dat extension and all files with a *-bytecode.jar suffix.
	files, err := getRegularFilesInDir(dependenciesDir, false)
	if err != nil {
		return err
	}
	for _, file := range files {
		if strings.HasSuffix(file, ".dat") || strings.HasSuffix(file, "-bytecode.jar") {
			source := path.Join(dependenciesDir, file)
			destination := path.Join(quarkusDir, file)
			if _, err = util.CopyFile(source, destination); err != nil {
				return err
			}
		}
	}

	return nil
}

func CopyLibFiles(dependenciesDir string, libDir string) error {
	// Create directory if one does not already exist
	if err := util.CreateDirectory(libDir); err != nil {
		return err
	}

	fileNames, err := getRegularFilesInDir(dependenciesDir, false)
	if err != nil {
		return err
	}

	for _, dependencyJar := range fileNames {
		source := path.Join(dependenciesDir, dependencyJar)
		destination := path.Join(libDir, dependencyJar)
		if _, err = util.CopyFile(source, destination); err != nil {
			return err
		}
	}

	return nil
}

func CopyAppFile(dependenciesDir string, appDir string) error {
	// Create directory if one does not already exist
	if err := util.CreateDirectory(appDir); err != nil {
		return err
	}

	fileNames, err := getRegularFilesInDir(dependenciesDir, false)
	if err != nil {
		return err
	}

	for _, dependencyJar := range fileNames {
		if strings.HasPrefix(dependencyJar, "camel-k-integration-") {
			source := path.Join(dependenciesDir, dependencyJar)
			destination := path.Join(appDir, dependencyJar)
			if _, err = util.CopyFile(source, destination); err != nil {
				return err
			}
		}
	}

	return nil
}
