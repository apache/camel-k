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
	"errors"
	"fmt"
	"os"
	"path"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/builder/runtime"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/scylladb/go-set/strset"
	"github.com/spf13/cobra"
)

var acceptedDependencyTypes = []string{"bom", "camel", "camel-k", "camel-quarkus", "mvn", "github"}

const (
	defaultRuntimeVersion         = "1.3.0"
	defaultCamelVersion           = "3.3.0"
	defaultWorkspaceDirectoryName = "workspace"
)

func newCmdInspect(rootCmdOptions *RootCmdOptions) (*cobra.Command, *inspectCmdOptions) {
	options := inspectCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "inspect [files to inspect]",
		Short:   "Generate dependencies list the given integration files.",
		Long:    "Output dependencies for a list of integration files. By default this command returns the top level dependencies only.",
		PreRunE: decode(&options),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	cmd.Flags().Bool("all-dependencies", false, "Compute and output transitive dependencies.")
	cmd.Flags().String("additional-dependencies", "", `Comma-separated list of additional top-level dependencies with the format:
		<type>:<dependency-name>
			where <type> is one of {`+strings.Join(acceptedDependencyTypes, "|")+`}.`)
	cmd.Flags().String("workspace", "", "Absolute path to workspace directory. Default: <kamel-invocation-directory>/workspace")
	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().String("dependencies-directory", "", "Absolute path to directory containing all dependencies. Default: <kamel-invocation-directory>/workspace/dependencies")

	return &cmd, &options
}

type inspectCmdOptions struct {
	*RootCmdOptions
	AllDependencies        bool   `mapstructure:"all-dependencies"`
	OutputFormat           string `mapstructure:"output"`
	AdditionalDependencies string `mapstructure:"additional-dependencies"`
	Workspace              string `mapstructure:"workspace"`
	DependenciesDirectory  string `mapstructure:"dependencies-directory"`
}

func (command *inspectCmdOptions) validate(args []string) error {
	// If no source files have been provided there is nothing to inspect.
	if len(args) == 0 {
		return errors.New("no integration files have been provided, nothing to inspect")
	}

	// Ensure source files exist.
	for _, arg := range args {
		// fmt.Printf("Validating file: %v\n", arg)
		fileExists, err := util.FileExists(arg)

		// Report any error.
		if err != nil {
			return err
		}

		// Signal file not found.
		if !fileExists {
			return errors.New("input file " + arg + " file does not exist")
		}
	}

	// Validate list of additional dependencies i.e. make sure that each dependency has
	// a valid type.
	if command.AdditionalDependencies != "" {
		additionalDependencies := strings.Split(command.AdditionalDependencies, ",")

		for _, dependency := range additionalDependencies {
			dependencyComponents := strings.Split(dependency, ":")

			TypeIsValid := false
			for _, dependencyType := range acceptedDependencyTypes {
				if dependencyType == dependencyComponents[0] {
					TypeIsValid = true
				}
			}

			if !TypeIsValid {
				return errors.New("Unexpected type for user-provided dependency: " + dependency + ", check command usage for valid format.")
			}
		}
	}

	// If provided, ensure that that the dependencies directory exists.
	if command.DependenciesDirectory != "" {
		dependenciesDirectoryExists, err := util.DirectoryExists(command.DependenciesDirectory)
		// Report any error.
		if err != nil {
			return err
		}

		// Signal file not found.
		if !dependenciesDirectoryExists {
			return errors.New("input file " + command.DependenciesDirectory + " file does not exist")
		}
	}

	return nil
}

func (command *inspectCmdOptions) run(args []string) error {
	// Attempt to reuse existing Camel catalog if one is present.
	catalog, err := camel.MainCatalog()
	if err != nil {
		return err
	}

	// Generate catalog if one was not found.
	if catalog == nil {
		catalog, err = generateCatalog()
		if err != nil {
			return err
		}
	}

	// Get top-level dependencies, this is the default behavior when no other options are provided.
	dependencies, err := getTopLevelDependencies(catalog, command.OutputFormat, args)
	if err != nil {
		return err
	}

	// Add additional user-provided dependencies.
	if command.AdditionalDependencies != "" {
		additionalDependencies := strings.Split(command.AdditionalDependencies, ",")
		for _, dependency := range additionalDependencies {
			dependencies = append(dependencies, dependency)
		}
	}

	// If --all-dependencies flag is set, add transitive dependencies.
	if command.AllDependencies {
		// Create workspace directory to hold all intermediate files.
		workspaceDirectory, err := getWorkspaceDirectory(command)
		if err != nil {
			return err
		}

		// The dependencies var will contain both top level and transitive dependencies.
		dependencies, err = getTransitiveDependencies(catalog, dependencies, workspaceDirectory)
		if err != nil {
			return err
		}
	}

	for _, dep := range dependencies {
		fmt.Printf("%v\n", dep)
	}

	return nil
}

func getTopLevelDependencies(catalog *camel.RuntimeCatalog, format string, args []string) ([]string, error) {
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

	if format != "" {
		err := printDependencies(format, dependencies)
		if err != nil {
			return []string{}, err
		}
	} else {
		// Print output in text form.
		for _, dep := range dependencies.List() {
			fmt.Printf("%v\n", dep)
		}
	}
	return dependencies.List(), nil
}

func generateCatalog() (*camel.RuntimeCatalog, error) {
	// A Camel catalog is requiref for this operatio.
	settings := ""
	mvn := v1.MavenSpec{
		LocalRepository: "",
	}
	runtime := v1.RuntimeSpec{
		Version:  defaults.DefaultRuntimeVersion,
		Provider: v1.RuntimeProviderMain,
	}
	providerDependencies := []maven.Dependency{}
	catalog, err := camel.GenerateCatalogCommon(settings, mvn, runtime, providerDependencies)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func printDependencies(format string, dependecies *strset.Set) error {
	switch format {
	case "yaml":
		data, err := util.DependenciesToYAML(dependecies.List())
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	case "json":
		data, err := util.DependenciesToJSON(dependecies.List())
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	default:
		return errors.New("unknown output format: " + format)
	}
	return nil
}

func getTransitiveDependencies(
	catalog *camel.RuntimeCatalog,
	dependencies []string,
	workspaceDirectory string) ([]string, error) {

	mvn := v1.MavenSpec{
		LocalRepository: "",
	}

	// Create Maven project.
	project := runtime.GenerateProjectCommon(defaultCamelVersion, defaultRuntimeVersion)

	// Inject dependencies into Maven project.
	err := builder.InjectDependenciesCommon(&project, dependencies, catalog)
	if err != nil {
		return []string{}, err
	}

	// Create local Maven context.
	mc := maven.NewContext(path.Join(workspaceDirectory, "maven"), project)
	mc.LocalRepository = mvn.LocalRepository
	mc.Timeout = mvn.GetTimeout().Duration

	// Compute dependencies.
	content, err := runtime.ComputeDependenciesCommon(mc, catalog.Runtime.Version)
	if err != nil {
		return nil, err
	}

	// Compose artifcats list.
	artifacts := []v1.Artifact{}
	artifacts, err = runtime.ProcessTransitiveDependencies(content, "dependencies")
	if err != nil {
		return nil, err
	}

	allDependencies := strset.New()
	for _, entry := range artifacts {
		allDependencies.Add(fmt.Sprintf("%s\n", entry.Location))
	}

	return allDependencies.List(), nil
}

func getWorkspaceDirectory(command *inspectCmdOptions) (string, error) {
	// Path to workspace directory.
	workspaceDirectory := command.Workspace
	if command.Workspace == "" {
		currentDirectory, err := os.Getwd()
		if err != nil {
			return "", err
		}
		workspaceDirectory = path.Join(currentDirectory, defaultWorkspaceDirectoryName)
	}

	// Create the workspace directory if it does not already exist.
	err := util.CreateDirectory(workspaceDirectory)
	if err != nil {
		return "", err
	}

	return workspaceDirectory, nil
}
