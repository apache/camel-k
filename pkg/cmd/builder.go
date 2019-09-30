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

func newCmdBuilder(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := builderCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:    "builder",
		Short:  "Run the Camel K builder",
		Long:   `Run the Camel K builder`,
		Hidden: true,
		Run:    impl.run,
	}

	cmd.Flags().StringVar(&impl.BuildName, "build-name", "", "The name of the build resource")

	return &cmd
}

type builderCmdOptions struct {
	*RootCmdOptions
	BuildName string
}

func (o *builderCmdOptions) run(_ *cobra.Command, _ []string) {
	builder.Run(o.Namespace, o.BuildName)
}
