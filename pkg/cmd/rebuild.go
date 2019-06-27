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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newCmdRebuild(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := rebuildCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "rebuild [integration]",
		Short: "Clear the state of integrations to rebuild them",
		Long:  `Clear the state of one or more integrations causing a rebuild.`,
		RunE:  options.rebuild,
	}

	return &cmd
}

type rebuildCmdOptions struct {
	*RootCmdOptions
}

func (o *rebuildCmdOptions) rebuild(_ *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}

	var integrations []v1alpha1.Integration
	if len(args) == 0 {
		if integrations, err = o.listAllIntegrations(c); err != nil {
			return err
		}
	} else {
		if integrations, err = o.getIntegrations(c, args); err != nil {
			return err
		}
	}

	if err = o.rebuildIntegrations(c, integrations); err != nil {
		return err
	}

	fmt.Printf("%d integrations have been rebuilt\n", len(integrations))
	return nil
}

func (o *rebuildCmdOptions) listAllIntegrations(c client.Client) ([]v1alpha1.Integration, error) {
	list := v1alpha1.NewIntegrationList()
	if err := c.List(o.Context, &k8sclient.ListOptions{Namespace: o.Namespace}, &list); err != nil {
		return nil, errors.Wrap(err, fmt.Sprintf("could not retrieve integrations from namespace %s", o.Namespace))
	}
	return list.Items, nil
}

func (o *rebuildCmdOptions) getIntegrations(c client.Client, names []string) ([]v1alpha1.Integration, error) {
	ints := make([]v1alpha1.Integration, 0, len(names))
	for _, n := range names {
		it := v1alpha1.NewIntegration(o.Namespace, n)
		key := k8sclient.ObjectKey{
			Name:      n,
			Namespace: o.Namespace,
		}
		if err := c.Get(o.Context, key, &it); err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("could not find integration %s in namespace %s", it.Name, o.Namespace))
		}
		ints = append(ints, it)
	}
	return ints, nil
}

func (o *rebuildCmdOptions) rebuildIntegrations(c client.Client, integrations []v1alpha1.Integration) error {
	for _, i := range integrations {
		it := i
		it.Status = v1alpha1.IntegrationStatus{}
		if err := c.Status().Update(o.Context, &it); err != nil {
			return errors.Wrap(err, fmt.Sprintf("could not rebuild integration %s in namespace %s", it.Name, o.Namespace))
		}
	}
	return nil
}
