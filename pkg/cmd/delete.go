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
	"strconv"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/spf13/cobra"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdDelete --
func newCmdDelete(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := deleteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "delete [integration1] [integration2] ...",
		Short:   "Delete integrations deployed on Kubernetes",
		PreRunE: decode(&impl),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := impl.validate(args); err != nil {
				return err
			}
			if err := impl.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().Bool("all", false, "Delete all integrations")

	return &cmd
}

type deleteCmdOptions struct {
	*RootCmdOptions
	DeleteAll bool `mapstructure:"all"`
}

func (command *deleteCmdOptions) validate(args []string) error {
	if command.DeleteAll && len(args) > 0 {
		return errors.New("invalid combination: both all flag and named integrations are set")
	}
	if !command.DeleteAll && len(args) == 0 {
		return errors.New("invalid combination: neither all flag nor named integrations are set")
	}

	return nil
}

func (command *deleteCmdOptions) run(args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}
	if len(args) != 0 && !command.DeleteAll {
		for _, arg := range args {
			name := kubernetes.SanitizeName(arg)
			err := DeleteIntegration(command.Context, c, name, command.Namespace)
			if err != nil {
				if k8errors.IsNotFound(err) {
					fmt.Println("Integration " + name + " not found. Skipped.")
				} else {
					return err
				}
			} else {
				fmt.Println("Integration " + name + " deleted")
			}
		}
	} else if command.DeleteAll {
		integrationList := v1alpha1.IntegrationList{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
				Kind:       v1alpha1.IntegrationKind,
			},
		}

		//Looks like Operator SDK doesn't support deletion of all objects with one command
		err := c.List(command.Context, &integrationList, k8sclient.InNamespace(command.Namespace))
		if err != nil {
			return err
		}
		for _, integration := range integrationList.Items {
			integration := integration // pin
			err := c.Delete(command.Context, &integration)
			if err != nil {
				return err
			}
		}
		if len(integrationList.Items) == 0 {
			fmt.Println("Nothing to delete")
		} else {
			fmt.Println(strconv.Itoa(len(integrationList.Items)) + " integration(s) deleted")
		}
	}

	return nil
}
