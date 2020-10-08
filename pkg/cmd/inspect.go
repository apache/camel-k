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
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
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

	// TODO: support the following options:
	cmd.Flags().Bool("all-dependencies", false, "Include both top level and transitive dependencies.")
	cmd.Flags().String("dependencies-directory", "", "If set, directory will contain all integration dependencies.")
	cmd.Flags().String("additional-dependencies", "", "Comma separated list of additional dependencies.")

	return &cmd, &options
}

type inspectCmdOptions struct {
	*RootCmdOptions
	AllDependencies        bool   `mapstructure:"all-dependencies"`
	DependenciesDirectory  string `mapstructure:"dependencies-directory"`
	AdditionalDependencies string `mapstructure:"additional-dependencies"`
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

	// TODO: validate list of additional dependencies i.e. make sure that each dependency is
	// in the correct format.

	return nil
}

func (command *inspectCmdOptions) run(args []string) error {
	// A Camel catalog is requiref for this operatio.
	settings := ""
	mvn := v1.MavenSpec{
		LocalRepository: "",
	}
	runtime := v1.RuntimeSpec{
		Version:  "1.3.0",
		Provider: v1.RuntimeProviderMain,
	}
	providerDependencies := []maven.Dependency{}
	catalog, err := camel.GenerateLocalCatalog(settings, mvn, runtime, providerDependencies)
	if err != nil {
		return err
	}

	// TODO: compression not supported for this command for now.
	compression := false

	// List of top-level dependencies.
	dependencies := strset.New()

	// Invoke the dependency inspector code for each source file.
	for _, source := range args {
		data, _, err := loadData(source, compression, compression)
		if err != nil {
			return err
		}

		sourceSpec := v1.SourceSpec{
			DataSpec: v1.DataSpec{
				Name:        path.Base(source),
				Content:     data,
				Compression: compression,
			},
		}

		// Extract list of top-level dependencies.
		metadata.Extract(catalog, sourceSpec)
		meta := metadata.Extract(catalog, sourceSpec)
		lang := sourceSpec.InferLanguage()

		// add auto-detected dependencies
		dependencies.Merge(meta.Dependencies)

		for loader, v := range catalog.Loaders {
			// add loader specific dependencies
			if sourceSpec.Loader != "" && sourceSpec.Loader == loader {
				dependencies.Add(fmt.Sprintf("mvn:%s/%s", v.GroupID, v.ArtifactID))

				for _, d := range v.Dependencies {
					dependencies.Add(fmt.Sprintf("mvn:%s/%s", d.GroupID, d.ArtifactID))
				}
			} else if sourceSpec.Loader == "" {
				// add language specific dependencies
				if util.StringSliceExists(v.Languages, string(lang)) {
					dependencies.Add(fmt.Sprintf("mvn:%s/%s", v.GroupID, v.ArtifactID))

					for _, d := range v.Dependencies {
						dependencies.Add(fmt.Sprintf("mvn:%s/%s", d.GroupID, d.ArtifactID))
					}
				}
			}
		}
	}

	for _, dep := range dependencies.List() {
		fmt.Printf("%v\n", dep)
	}

	return nil
}
