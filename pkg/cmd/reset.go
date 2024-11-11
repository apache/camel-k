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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"

	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/runtime/schema"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newCmdReset(rootCmdOptions *RootCmdOptions) (*cobra.Command, *resetCmdOptions) {
	options := resetCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "reset",
		Short:   "Reset the Camel K installation",
		Long:    `Reset the Camel K installation by deleting everything except current integration profile.`,
		PreRunE: decode(&options, options.Flags),
		Run:     options.reset,
	}

	cmd.Flags().Bool("skip-kits", false, "Do not delete the integration kits")
	cmd.Flags().Bool("skip-integrations", false, "Do not delete the integrations")
	cmd.Flags().Bool("skip-bindings", false, "Do not delete the bindings/pipes")

	return &cmd, &options
}

type resetCmdOptions struct {
	*RootCmdOptions
	SkipKits         bool `mapstructure:"skip-kits"`
	SkipIntegrations bool `mapstructure:"skip-integrations"`
	SkipBindings     bool `mapstructure:"skip-bindings"`
}

func (o *resetCmdOptions) reset(cmd *cobra.Command, _ []string) {
	c, err := o.GetCmdClient()
	if err != nil {
		fmt.Fprint(cmd.ErrOrStderr(), err)
		return
	}

	var n int
	if !o.SkipBindings {
		if n, err = o.deleteAllPipes(c); err != nil {
			fmt.Fprint(cmd.ErrOrStderr(), err)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), n, "pipes deleted from namespace", o.Namespace)
	}

	if !o.SkipIntegrations {
		if n, err = o.deleteAllIntegrations(c); err != nil {
			fmt.Fprint(cmd.ErrOrStderr(), err)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), n, "integrations deleted from namespace", o.Namespace)
	}

	if !o.SkipKits {
		if n, err = o.deleteAllIntegrationKits(c); err != nil {
			fmt.Fprint(cmd.ErrOrStderr(), err)
			return
		}
		fmt.Fprintln(cmd.OutOrStdout(), n, "integration kits deleted from namespace", o.Namespace)
	}
}

func (o *resetCmdOptions) deleteAllIntegrations(c client.Client) (int, error) {
	list := v1.NewIntegrationList()
	if err := c.List(o.Context, &list, k8sclient.InNamespace(o.Namespace)); err != nil {
		return 0, fmt.Errorf("could not retrieve integrations from namespace %s: %w", o.Namespace, err)
	}
	for _, i := range list.Items {
		it := i
		if isIntegrationOwned(it) {
			// Deleting it directly is ineffective, deleting the controller will delete it
			continue
		}
		if err := c.Delete(o.Context, &it); err != nil {
			return 0, fmt.Errorf("could not delete integration %s from namespace %s: %w", it.Name, it.Namespace, err)
		}
	}
	return len(list.Items), nil
}

func (o *resetCmdOptions) deleteAllIntegrationKits(c client.Client) (int, error) {
	list := v1.NewIntegrationKitList()
	if err := c.List(o.Context, &list, k8sclient.InNamespace(o.Namespace)); err != nil {
		return 0, fmt.Errorf("could not retrieve integration Kits from namespace %s: %w", o.Namespace, err)
	}
	for _, i := range list.Items {
		kit := i
		if err := c.Delete(o.Context, &kit); err != nil {
			return 0, fmt.Errorf("could not delete integration kit %s from namespace %s: %w", kit.Name, kit.Namespace, err)
		}
	}
	return len(list.Items), nil
}

func (o *resetCmdOptions) deleteAllPipes(c client.Client) (int, error) {
	list := v1.NewPipeList()
	if err := c.List(o.Context, &list, k8sclient.InNamespace(o.Namespace)); err != nil {
		return 0, fmt.Errorf("could not retrieve Pipes from namespace %s: %w", o.Namespace, err)
	}
	for _, i := range list.Items {
		klb := i
		if err := c.Delete(o.Context, &klb); err != nil {
			return 0, fmt.Errorf("could not delete Pipe %s from namespace %s: %w", klb.Name, klb.Namespace, err)
		}
	}
	return len(list.Items), nil
}

func isIntegrationOwned(it v1.Integration) bool {
	for _, ref := range it.OwnerReferences {
		gv, err := schema.ParseGroupVersion(ref.APIVersion)
		if err != nil {
			continue
		}
		if gv.Group == v1.SchemeGroupVersion.Group && ref.BlockOwnerDeletion != nil && *ref.BlockOwnerDeletion {
			return true
		}
	}
	return false
}
