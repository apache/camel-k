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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func newCmdRebuild(rootCmdOptions *RootCmdOptions) (*cobra.Command, *rebuildCmdOptions) {
	options := rebuildCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "rebuild [integration1] [integration2] ...",
		Short:   "Clear the state of integrations to rebuild them.",
		Long:    `Clear the state of one or more integrations causing a rebuild. Rebuild always targets Integration CR, the operator is in charge to apply any change to the related bindings resources (if any).`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			return options.run(cmd, args)
		},
	}

	cmd.Flags().Bool("all", false, "Rebuild all integrations")

	return &cmd, &options
}

type rebuildCmdOptions struct {
	*RootCmdOptions
	RebuildAll bool `mapstructure:"all"`
}

func (o *rebuildCmdOptions) validate(args []string) error {
	if o.RebuildAll && len(args) > 0 {
		return errors.New("invalid combination: --all flag is set and at least one integration name is provided")
	}
	if !o.RebuildAll && len(args) == 0 {
		return errors.New("invalid combination: provide one or several integration names or set --all flag for all integrations")
	}

	return nil
}

func (o *rebuildCmdOptions) run(cmd *cobra.Command, args []string) error {
	errKlbs := o.rebuildBindingType(cmd, args)
	errIts := o.rebuildIntegrationType(cmd, args)

	if errIts != nil && errKlbs != nil {
		return errors.Wrap(errIts, errKlbs.Error())
	}

	return nil
}

func (o *rebuildCmdOptions) rebuildBindingType(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	var bindings []v1.Binding
	if o.RebuildAll {
		if bindings, err = o.listAllBindings(c); err != nil {
			return err
		}
	} else if len(args) > 0 {
		if bindings, err = o.getBindings(c, args); err != nil {
			return err
		}
	}

	if err = o.rebuildBindings(c, bindings); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), len(bindings), "bindings have been rebuilt")
	return nil
}

func (o *rebuildCmdOptions) listAllBindings(c client.Client) ([]v1.Binding, error) {
	list := v1.NewBindingList()
	if err := c.List(o.Context, &list, k8sclient.InNamespace(o.Namespace)); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve bindings from namespace %s", o.Namespace))
	}
	return list.Items, nil
}

func (o *rebuildCmdOptions) getBindings(c client.Client, names []string) ([]v1.Binding, error) {
	klbs := make([]v1.Binding, 0, len(names))
	for _, n := range names {
		klb := v1.NewBinding(o.Namespace, n)
		key := k8sclient.ObjectKey{
			Name:      n,
			Namespace: o.Namespace,
		}
		if err := c.Get(o.Context, key, &klb); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not find binding %s in namespace %s", klb.Name, o.Namespace))
		}
		klbs = append(klbs, klb)
	}
	return klbs, nil
}

func (o *rebuildCmdOptions) rebuildBindings(c k8sclient.StatusClient, bindings []v1.Binding) error {
	for _, i := range bindings {
		klb := i
		klb.Status = v1.BindingStatus{}
		if err := c.Status().Update(o.Context, &klb); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not rebuild binding %s in namespace %s", klb.Name, o.Namespace))
		}
	}
	return nil
}

func (o *rebuildCmdOptions) rebuildIntegrationType(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	var integrations []v1.Integration
	if o.RebuildAll {
		if integrations, err = o.listAllIntegrations(c); err != nil {
			return err
		}
	} else if len(args) > 0 {
		if integrations, err = o.getIntegrations(c, args); err != nil {
			return err
		}
	}

	if err = o.rebuildIntegrations(c, integrations); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), len(integrations), "integrations have been rebuilt")
	return nil
}

func (o *rebuildCmdOptions) listAllIntegrations(c client.Client) ([]v1.Integration, error) {
	list := v1.NewIntegrationList()
	// Integrations controlled by Bindings are not included
	excludeItsFromKlbs, err := labels.NewRequirement(kubernetes.CamelCreatorLabelKind, selection.NotEquals, []string{
		"Binding",
	})
	if err != nil {
		return list.Items, err
	}
	if err := c.List(o.Context, &list,
		k8sclient.InNamespace(o.Namespace),
		k8sclient.MatchingLabelsSelector{
			Selector: labels.NewSelector().Add(*excludeItsFromKlbs),
		},
	); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve integrations from namespace %s", o.Namespace))
	}
	return list.Items, nil
}

func (o *rebuildCmdOptions) getIntegrations(c client.Client, names []string) ([]v1.Integration, error) {
	ints := make([]v1.Integration, 0, len(names))
	for _, n := range names {
		it := v1.NewIntegration(o.Namespace, n)
		key := k8sclient.ObjectKey{
			Name:      n,
			Namespace: o.Namespace,
		}
		if err := c.Get(o.Context, key, &it); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not find integration %s in namespace %s", it.Name, o.Namespace))
		}
		// Integrations controlled by Bindings are not included
		if it.Labels[kubernetes.CamelCreatorLabelKind] != "Binding" {
			ints = append(ints, it)
		}
	}
	return ints, nil
}

func (o *rebuildCmdOptions) rebuildIntegrations(c k8sclient.StatusClient, integrations []v1.Integration) error {
	for _, i := range integrations {
		it := i
		it.Status = v1.IntegrationStatus{}
		if err := c.Status().Update(o.Context, &it); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not rebuild integration %s in namespace %s", it.Name, o.Namespace))
		}
	}
	return nil
}
