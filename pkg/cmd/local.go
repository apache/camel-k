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
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

// NewCmdLocal -- Add local kamel subcommand with several other subcommands of its own.
func newCmdLocal(rootCmdOptions *RootCmdOptions) *cobra.Command {
	cmd := cobra.Command{
		Use:   "local [sub-command]",
		Short: "Perform integration actions locally.",
		Long:  `Perform integration actions locally given a set of input integration files.`,
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}

	return &cmd
}

func addLocalSubCommands(cmd *cobra.Command, options *RootCmdOptions) error {
	var localCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "local" {
			localCmd = c
			break
		}
	}

	if localCmd == nil {
		return errors.New("could not find any configured local command")
	}

	localCmd.AddCommand(cmdOnly(newCmdLocalCreate(options)))
	localCmd.AddCommand(cmdOnly(newCmdLocalInspect(options)))
	localCmd.AddCommand(cmdOnly(newCmdLocalRun(options)))

	return nil
}
