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

	"github.com/spf13/cobra"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func newCmdUndeploy(rootCmdOptions *RootCmdOptions) (*cobra.Command, *undeployCmdOptions) {
	options := undeployCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "undeploy [integration1] [integration2] ...",
		Short:   "Undeploy one or more integrations previously deployed.",
		Long:    `Clear the state of one or more integrations causing them to move back to a Build Complete status.`,
		PreRunE: decode(&options, options.Flags),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}

			return options.run(cmd, args)
		},
	}

	return &cmd, &options
}

type undeployCmdOptions struct {
	*RootCmdOptions
}

func (o *undeployCmdOptions) validate(args []string) error {
	if len(args) == 0 {
		return errors.New("undeploy requires an Integration name argument")
	}

	return nil
}

func (o *undeployCmdOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	var integrations []v1.Integration
	if len(args) > 0 {
		if integrations, err = getIntegrations(o.Context, c, args, o.Namespace); err != nil {
			return err
		}
	}

	undeployed, err := o.undeployIntegrations(cmd, c, integrations)
	// We print the number of undeployed integrations anyway (they could have been correctly processed)
	fmt.Fprintln(cmd.OutOrStdout(), undeployed, "integrations have been undeployed")

	return err
}

func (o *undeployCmdOptions) undeployIntegrations(cmd *cobra.Command, c k8sclient.StatusClient, integrations []v1.Integration) (int, error) {
	undeployed := 0
	for _, i := range integrations {
		if i.Status.Phase != v1.IntegrationPhaseRunning {
			fmt.Fprintf(cmd.OutOrStdout(),
				"warning: could not undeploy integration %s, it is not in status %s\n",
				i.Name, v1.IntegrationPhaseRunning)

			continue
		}
		if i.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] != "true" {
			fmt.Fprintf(cmd.OutOrStdout(),
				"warning: could not undeploy integration %s, it is not annotated with %s=true\n",
				i.Name, v1.IntegrationDontRunAfterBuildAnnotation)

			continue
		}
		it := i
		it.Status.Phase = v1.IntegrationPhaseInitialization
		if err := c.Status().Update(o.Context, &it); err != nil {
			return undeployed, fmt.Errorf("could not undeploy %s in namespace %s: %w", it.Name, o.Namespace, err)
		}
		undeployed++
	}

	return undeployed, nil
}
