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
	"github.com/spf13/cobra"
)

func newCmdKamelet(rootCmdOptions *RootCmdOptions) *cobra.Command {
	cmd := cobra.Command{
		Use:   "kamelet",
		Short: "Configure a Kamelet",
		Long:  `Configure a Kamelet.`,
	}

	cmd.AddCommand(cmdOnly(newKameletGetCmd(rootCmdOptions)))
	cmd.AddCommand(cmdOnly(newKameletDeleteCmd(rootCmdOptions)))
	cmd.AddCommand(cmdOnly(newKameletAddRepoCmd(rootCmdOptions)))
	cmd.AddCommand(cmdOnly(newKameletRemoveRepoCmd(rootCmdOptions)))

	return &cmd
}
