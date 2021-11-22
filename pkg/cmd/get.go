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
	"text/tabwriter"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

type getCmdOptions struct {
	*RootCmdOptions
}

func newCmdGet(rootCmdOptions *RootCmdOptions) (*cobra.Command, *getCmdOptions) {
	options := getCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "get [integration]",
		Short:   "Get integrations deployed on Kubernetes",
		Long:    `Get the status of integrations deployed on Kubernetes.`,
		PreRunE: decode(&options),
		RunE:    options.run,
	}

	return &cmd, &options
}

func (o *getCmdOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	integrationList := v1.IntegrationList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       "Integration",
		},
	}

	namespace := o.Namespace

	options := []k8sclient.ListOption{
		k8sclient.InNamespace(namespace),
	}
	if len(args) == 1 {
		options = append(options, k8sclient.MatchingFields{
			"metadata.name": args[0],
		})
	}

	err = c.List(o.Context, &integrationList, options...)
	if err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "NAME\tPHASE\tKIT")
	for _, integration := range integrationList.Items {
		kit := ""
		if integration.Status.IntegrationKit != nil {
			ns := integration.GetIntegrationKitNamespace(nil)
			kit = fmt.Sprintf("%s/%s", ns, integration.Status.IntegrationKit.Name)
		}
		fmt.Fprintf(w, "%s\t%s\t%s\n", integration.Name, string(integration.Status.Phase), kit)
	}

	return w.Flush()
}
