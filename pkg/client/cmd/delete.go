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
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/cobra"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCmdDelete --
func newCmdDelete(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := deleteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "delete [integration1] [integration2] ...",
		Short: "Delete integrations deployed on Kubernetes",
		RunE:  options.run,
	}
	cmd.Flags().BoolVar(&options.deleteAll, "all", false, "Delete all integrations")
	cmd.ParseFlags(os.Args)

	return &cmd
}

type deleteCmdOptions struct {
	*RootCmdOptions
	deleteAll bool
}

func (o *deleteCmdOptions) run(cmd *cobra.Command, args []string) error {
	namespace := o.Namespace

	integrationList := v1alpha1.IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.IntegrationKind,
		},
	}

	if len(args) != 0 && !o.deleteAll {
		i := 0
		for i < len(args) {
			integration := v1alpha1.Integration{
				TypeMeta: metav1.TypeMeta{
					Kind:       v1alpha1.IntegrationKind,
					APIVersion: v1alpha1.SchemeGroupVersion.String(),
				},
				ObjectMeta: metav1.ObjectMeta{
					Namespace: namespace,
					Name:      args[i],
				},
			}

			err := sdk.Delete(&integration)
			if err != nil {
				if k8errors.IsNotFound(err) {
					fmt.Println("Integration " + integration.GetName() + " not found. Skipped.")
				} else {
					return err
				}
			} else {
				fmt.Println("Integration " + integration.GetName() + " deleted")
			}
			i++
		}
	} else if o.deleteAll {
		//Looks like Operator SDK doesn't support deletion of all objects with one command
		err := sdk.List(namespace, &integrationList)
		if err != nil {
			return err
		}
		for _, integration := range integrationList.Items {
			err := sdk.Delete(&integration)
			if err != nil {
				return err
			}
		}
		if len(integrationList.Items) == 0 {
			fmt.Println("Nothing to delete")
		} else {
			fmt.Println(strconv.Itoa(len(integrationList.Items)) + " integration(s) deleted")
		}
	} else {
		err := errors.New("The integration name(s) or --all option must be specified")
		return err

	}

	return nil
}
