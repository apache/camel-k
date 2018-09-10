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
	"text/tabwriter"
	"os"
	"fmt"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

type GetCmdOptions struct {
	*RootCmdOptions
}

func NewCmdGet(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := GetCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "get",
		Short: "Get all integrations deployed on Kubernetes",
		Long:  `Get the status of all integrations deployed on on Kubernetes.`,
		RunE:  options.run,
	}

	return &cmd
}

func (o *GetCmdOptions) run(cmd *cobra.Command, args []string) error {
	integrationList := v1alpha1.IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "Integration",
		},
	}

	namespace := o.Namespace

	err := sdk.List(namespace, &integrationList)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(os.Stdout, 0, 8, 0, '\t', 0)
	fmt.Fprintln(w, "NAME\tSTATUS")
	for _, integration := range integrationList.Items {
		fmt.Fprintln(w, integration.Name+"\t"+string(integration.Status.Phase))
	}
	w.Flush()

	return nil
}
