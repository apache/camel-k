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
	"github.com/apache/camel-k/pkg/cmd/builder"
	"github.com/spf13/cobra"
)

const builderCommand = "builder"

func newCmdBuilder(rootCmdOptions *RootCmdOptions) (*cobra.Command, *builderCmdOptions) {
	options := builderCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     builderCommand,
		Short:   "Run the Camel K builder",
		Long:    `Run the Camel K builder`,
		Hidden:  true,
		PreRunE: decode(&options),
		Run:     options.run,
	}

	cmd.Flags().String("build-name", "", "The name of the build resource")
	cmd.Flags().String("task-name", "", "The name of task to execute")

	return &cmd, &options
}

type builderCmdOptions struct {
	*RootCmdOptions
	BuildName string `mapstructure:"build-name"`
	TaskName  string `mapstructure:"task-name"`
}

func (o *builderCmdOptions) run(_ *cobra.Command, _ []string) {
	builder.Run(o.Namespace, o.BuildName, o.TaskName)
}
