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
	"path"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/maven"
	"github.com/scylladb/go-set/strset"
	"github.com/spf13/cobra"
)

func newCmdInspect(rootCmdOptions *RootCmdOptions) (*cobra.Command, *inspectCmdOptions) {
	options := inspectCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "inspect [files to inspect]",
		Short:   "Generate dependencies list the given integration files.",
		Long:    `Compute and emit the dependencies for a list of integration files.`,
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
	}

	cmd.Flags().StringP("output", "o", "", "Output format. One of: json|yaml")

	return &cmd, &options
}

type inspectCmdOptions struct {
	*RootCmdOptions
	OutputFormat string `mapstructure:"output"`
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

	// TODO: validate list of additional dependencies i.e. make sure that each dependency is
	// in the correct format.

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

	// List of top-level dependencies.
	dependencies := strset.New()

	// Invoke the dependency inspector code for each source file.
	for _, source := range args {
		data, _, err := loadContent(source, false, false)
		if err != nil {
			return err
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

	if command.OutputFormat != "" {
		err := printDependencies(command.OutputFormat, dependencies)
		if err != nil {
			return err
		}
	} else {
		// Print output in text form.
		for _, dep := range dependencies.List() {
			fmt.Printf("%v\n", dep)
		}
	}

	return nil
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
	catalog, err := camel.GenerateLocalCatalog(settings, mvn, runtime, providerDependencies)
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
