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
	"fmt"
	"sort"
	"strings"

	"github.com/spf13/cobra"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/camel"
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
        local type_list="mvn:"
        COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
		compopt -o nospace
        ;;
    f*)
        local type_list="file:"
        COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
		compopt -o nospace
        ;;
    *)
        local type_list="camel: mvn: file:"
        COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
	    compopt -o nospace
    esac
}

__kamel_traits() {
    local type_list="` + strings.Join(trait.NewCatalog(nil).ComputeTraitsProperties(), " ") + `"
    COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
    compopt -o nospace
}

__kamel_languages() {
    local type_list="js groovy kotlin java xml"
    COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
}

__kamel_deletion_policy() {
    local type_list="owner label"
    COMPREPLY=( $( compgen -W "${type_list}" -- "$cur") )
}

__kamel_kubectl_get_servicebinding() {
    local template
    local template_gvkn
    local kubectl_out
    local service_names
    local services_list
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"
    template_gvkn="{{ range .items  }}{{ .kind  }}/{{ .apiVersion  }}/{{ .metadata.name }} {{ end }}" 
    if kubectl_out=$(kubectl get -o template --template="${template}" ${namespace_condition} crd -l service.binding/provisioned-service=true 2>/dev/null); then
        kubectl_out="${kubectl_out// /,}"
        service_names="${kubectl_out}servicebinding"
        if kubectl_out=$(kubectl get -o template --template="${template_gvkn}" ${namespace_condition} ${service_names} 2>/dev/null); then
            for resource in  $kubectl_out
            do
               name=$(echo ${resource} | cut -d'/' -f 4)
               version=$(echo ${resource} | cut -d'/' -f 3)
               group=$(echo ${resource} | cut -d'/' -f 2)
               kind=$(echo ${resource} | cut -d'/' -f 1)
               services_list="${services_list} ${group}/${version}:${kind}:${name}"
            done
            COMPREPLY=( $( compgen -W "${services_list[*]}" -- "$cur" ) )
        fi
    fi
}

__kamel_kubectl_get_configmap() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" ${namespace_condition} configmap 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_secret() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" ${namespace_condition} secret 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_integrations() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" ${namespace_condition} integrations 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_integrationkits() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" ${namespace_condition} integrationkits 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_non_platform_integrationkits() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"
    label_condition="camel.apache.org/kit.type!=platform"

    if kubectl_out=$(kubectl get -l ${label_condition} -o template --template="${template}" ${namespace_condition} integrationkits 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_kamelets() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"

    if kubectl_out=$(kubectl get -o template --template="${template}" ${namespace_condition} kamelets 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__kamel_kubectl_get_non_bundled_non_readonly_kamelets() {
    local template
    local kubectl_out
    local namespace_condition
    
    if command -v awk &> /dev/null ; then
        local namespace_config=$(${COMP_WORDS[0]} config --list | awk '/default-namespace/{print $2}')
        if [ ! -z $namespace_config ]; then
            namespace_condition=$(echo "--namespace ${namespace_config}")
        fi
    fi
    
    local namespace_flag=$(echo "${flaghash['-n']}${flaghash['--namespace']}")
    if [ ! -z $namespace_flag ]; then
        namespace_condition=$(echo "--namespace ${namespace_flag}")
    fi

    template="{{ range .items  }}{{ .metadata.name }} {{ end }}"
    label_conditions="camel.apache.org/kamelet.bundled=false,camel.apache.org/kamelet.readonly=false"

    if kubectl_out=$(kubectl get -l ${label_conditions} -o template --template="${template}" ${namespace_condition} kamelets 2>/dev/null); then
        COMPREPLY=( $( compgen -W "${kubectl_out}" -- "$cur" ) )
    fi
}

__custom_func() {
    case ${last_command} in
        kamel_describe_integration)
            __kamel_kubectl_get_integrations
            return
            ;;
        kamel_describe_kit)
            __kamel_kubectl_get_integrationkits
            return
            ;;
        kamel_describe_kamelet)
            __kamel_kubectl_get_kamelets
            return
            ;;
        kamel_delete)
            __kamel_kubectl_get_integrations
            return
            ;;
        kamel_log)
            __kamel_kubectl_get_integrations
            return
            ;;
        kamel_get)
            __kamel_kubectl_get_integrations
            return
            ;;
        kamel_kit_delete)
            __kamel_kubectl_get_non_platform_integrationkits
            return
            ;;
        kamel_kit_get)
            __kamel_kubectl_get_integrationkits
            return
            ;;
        kamel_kamelet_delete)
            __kamel_kubectl_get_non_bundled_non_readonly_kamelets
            return
            ;;
        *)
            ;;
    esac
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
		Run: func(_ *cobra.Command, _ []string) {
			err := root.GenBashCompletion(root.OutOrStdout())
			if err != nil {
				fmt.Fprint(root.ErrOrStderr(), err.Error())
			}
		},
		Annotations: map[string]string{
			offlineCommandLabel: "true",
		},
	}
}

func configureKnownBashCompletions(command *cobra.Command) {
	configureBashAnnotationForFlag(
		command,
		"dependency",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_dependency_type"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"configmap",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_kubectl_get_configmap"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"secret",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_kubectl_get_secret"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"kit",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_kubectl_get_non_platform_integrationkits"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"language",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_languages"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"trait",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_traits"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"deletion-policy",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_deletion_policy"},
		},
	)
	configureBashAnnotationForFlag(
		command,
		"connect",
		map[string][]string{
			cobra.BashCompCustom: {"__kamel_kubectl_get_servicebinding"},
		},
	)
}

func configureBashAnnotationForFlag(command *cobra.Command, flagName string, annotations map[string][]string) {
	if flag := command.Flag(flagName); flag != nil {
		flag.Annotations = annotations
	}
}

func computeCamelDependencies() string {
	catalog, err := camel.DefaultCatalog()
	if err != nil || catalog == nil {
		catalog = camel.NewRuntimeCatalog(v1.CamelCatalog{}.Spec)
	}

	results := make([]string, 0, len(catalog.Artifacts)+len(catalog.Loaders))
	for a := range catalog.Artifacts {
		// skipping camel-k-* and other artifacts as they may not be useful for cli completion
		if strings.HasPrefix(a, "camel-quarkus-") {
			results = append(results, camel.NormalizeDependency(a))
		}
	}
	for _, l := range catalog.Loaders {
		if strings.HasPrefix(l.ArtifactID, "camel-quarkus-") {
			results = append(results, camel.NormalizeDependency(l.ArtifactID))
		}
	}
	sort.Strings(results)

	return strings.Join(results, " ")
}
