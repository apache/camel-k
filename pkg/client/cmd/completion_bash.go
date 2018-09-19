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

	"github.com/apache/camel-k/pkg/util/camel"

	"github.com/spf13/cobra"
)

// ******************************
//
//
//
// ******************************

const bashCompletionCmdLongDescription = `
To load completion run

. <(kamel completion bash)

To configure your bash shell to load completions for each session add to your bashrc

# ~/.bashrc or ~/.profile
. <(kamel completion bash)
`

var bashCompletionFunction = `
__kamel_dependency_type() {
    case ${cur} in
    c*)
        local type_list="` + computeCamelDependencies() + `"
        COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
        ;;
    m*)
        local type_list="mvn:""
        COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
		compopt -o nospace
        ;;
    f*)
        local type_list="file:""
        COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
		compopt -o nospace
        ;;
    *)
        local type_list="camel: mvn: file:"
        COMPREPLY=( $( compgen -W "camel mvn: file:" -- "$cur") )
	    compopt -o nospace
    esac
}

__kamel_kubectl_get_configmap() {
    local template
    local kubectl_out

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" configmap 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_secret() {
    local template
    local kubectl_out

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" secret 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}
`

// ******************************
//
// COMMAND
//
// ******************************

func newCmdCompletionBash(root *cobra.Command) *cobra.Command {
	return &cobra.Command{
		Use:   "bash",
		Short: "Generates bash completion scripts",
		Long:  bashCompletionCmdLongDescription,
		Run: func(cmd *cobra.Command, args []string) {
			root.GenBashCompletion(os.Stdout)
		},
	}
}

func configureKnownBashCompletions(command *cobra.Command) {
	// completion support
	dependencyFlag := command.Flag("dependency")
	if dependencyFlag != nil {
		dependencyFlag.Annotations = map[string][]string{
			cobra.BashCompCustom: {"__kamel_dependency_type"},
		}
	}

	configMapFlag := command.Flag("configmap")
	configMapFlag.Annotations = map[string][]string{
		cobra.BashCompCustom: {"__kamel_kubectl_get_configmap"},
	}

	secretFlag := command.Flag("secret")
	secretFlag.Annotations = map[string][]string{
		cobra.BashCompCustom: {"__kamel_kubectl_get_secret"},
	}
}

func computeCamelDependencies() string {
	result := ""

	for k := range camel.Runtime.Artifact {
		if result != "" {
			result = result + " " + k
		} else {
			result = k
		}
	}

	return result
}
