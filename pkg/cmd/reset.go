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

func newCmdReset(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := resetCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "reset",
		Short: "Reset the Camel K installation",
		Long:  `Reset the Camel K installation by deleting everything except current platform configuration.`,
		Run:   options.reset,
	}

	return &cmd
}

type resetCmdOptions struct {
	*RootCmdOptions
}

func (o *resetCmdOptions) reset(_ *cobra.Command, _ []string) {
	c, err := o.GetCmdClient()
	if err != nil {
		fmt.Print(err)
		return
	}
	var n int
	if n, err = o.deleteAllIntegrations(c); err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("%d integrations deleted from namespace %s\n", n, o.Namespace)

	if n, err = o.deleteAllIntegrationContexts(c); err != nil {
		fmt.Print(err)
		return
	}
	fmt.Printf("%d integration contexts deleted from namespace %s\n", n, o.Namespace)

	if err = o.resetIntegrationPlatform(c); err != nil {
		fmt.Print(err)
		return
	}

	fmt.Println("Camel K platform has been reset successfully!")
}

func (o *resetCmdOptions) deleteAllIntegrations(c client.Client) (int, error) {
	list := v1alpha1.NewIntegrationList()
	if err := c.List(o.Context, &k8sclient.ListOptions{Namespace: o.Namespace}, &list); err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("could not retrieve integrations from namespace %s", o.Namespace))
	}
	for _, i := range list.Items {
		it := i
		if err := c.Delete(o.Context, &it); err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("could not delete integration %s from namespace %s", it.Name, it.Namespace))
		}
	}
	return len(list.Items), nil
}

func (o *resetCmdOptions) deleteAllIntegrationContexts(c client.Client) (int, error) {
	list := v1alpha1.NewIntegrationContextList()
	if err := c.List(o.Context, &k8sclient.ListOptions{Namespace: o.Namespace}, &list); err != nil {
		return 0, errors.Wrap(err, fmt.Sprintf("could not retrieve integration contexts from namespace %s", o.Namespace))
	}
	for _, i := range list.Items {
		ictx := i
		if err := c.Delete(o.Context, &ictx); err != nil {
			return 0, errors.Wrap(err, fmt.Sprintf("could not delete integration context %s from namespace %s", ictx.Name, ictx.Namespace))
		}
	}
	return len(list.Items), nil
}

func (o *resetCmdOptions) resetIntegrationPlatform(c client.Client) error {
	list := v1alpha1.NewIntegrationPlatformList()
	if err := c.List(o.Context, &k8sclient.ListOptions{Namespace: o.Namespace}, &list); err != nil {
		return errors.Wrap(err, fmt.Sprintf("could not retrieve integration platform from namespace %s", o.Namespace))
	}
	if len(list.Items) > 1 {
		return errors.New(fmt.Sprintf("expected 1 integration platform in the namespace, found: %d", len(list.Items)))
	} else if len(list.Items) == 0 {
		return errors.New("no integration platforms found in the namespace: run \"kamel install\" to install the platform")
	}
	platform := list.Items[0]
	// Let's reset the status
	platform.Status = v1alpha1.IntegrationPlatformStatus{}
	return c.Update(o.Context, &platform)
}
