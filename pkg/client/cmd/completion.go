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
	"os"

	"github.com/spf13/cobra"
)

const bashCompletionCmdLongDescription = `
To load completion run

. <(kamel completion)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(kamel completion)
`

const zshCompletionCmdLongDescription = `
To configure your zsh shell to load completions for each session add to your zshrc

if [ $commands[kamel] ]; then
  source <(kamel completion zsh)
fi
`

// NewCmdCompletion --
func NewCmdCompletion(root *cobra.Command) *cobra.Command {
	completion := cobra.Command{
		Use:   "completion",
		Short: "Generates completion scripts",
	}

	completion.AddCommand(&cobra.Command{
		Use:   "bash",
		Short: "Generates bash completion scripts",
		Long:  bashCompletionCmdLongDescription,
		Run: func(cmd *cobra.Command, args []string) {
			root.GenBashCompletion(os.Stdout)
		},
	})

	completion.AddCommand(&cobra.Command{
		Use:   "zsh",
		Short: "Generates zsh completion scripts",
		Long:  zshCompletionCmdLongDescription,
		Run: func(cmd *cobra.Command, args []string) {
			root.GenZshCompletion(os.Stdout)
		},
	})

	return &completion
}
