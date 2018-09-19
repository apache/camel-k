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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/spf13/cobra"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// NewCmdDelete --
func newCmdDelete(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := deleteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "delete",
		Short: "Delete integrations deployed on Kubernetes",
		RunE:  options.run,
	}

	cmd.Flags().StringVar(&options.integrationName, "name", "", "The integration name")
	cmd.Flags().BoolVar(&options.deleteAll, "all", false, "Delete all integrations")
	cmd.ParseFlags(os.Args)

	return &cmd
}

type deleteCmdOptions struct {
	*RootCmdOptions
	integrationName string
	deleteAll       bool
}

func (o *deleteCmdOptions) run(cmd *cobra.Command, args []string) error {
	namespace := o.Namespace

	integrationList := v1alpha1.IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       v1alpha1.IntegrationKind,
		},
	}

	integration := v1alpha1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind:       v1alpha1.IntegrationKind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      o.integrationName,
		},
	}

	if o.integrationName != "" && !o.deleteAll {
		err := sdk.Delete(&integration)
		if err != nil {
			return err
		}

		fmt.Println("Integration " + integration.GetName() + " deleted")
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
	} else {
		err := errors.New("An integration name or --all option must be specified")
		return err
	}

	return nil
}
