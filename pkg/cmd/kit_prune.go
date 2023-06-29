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
	"context"
	"fmt"

	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/spf13/cobra"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func newKitPruneCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kitPruneCommandOptions) {
	options := kitPruneCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "prune",
		Short: "Delete unused Integration Kits",
		Long:  `Delete Integration Kits that are not referenced by Integrations.`,
		PreRunE: func(cmd *cobra.Command, args []string) error {
			if err := decode(&options)(cmd, args); err != nil {
				return err
			}
			return options.preRun(cmd, args)
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			return options.run(cmd, args)
		},
	}
	cmd.Flags().BoolP("assumeyes", "y", false, "Do not ask user to confirm Kits to be deleted")
	cmd.Flags().BoolP("dry-run", "d", false, "Only list Kits to be deleted without removing them")

	return &cmd, &options
}

type kitPruneCommandOptions struct {
	// TODO: add option to list Namespaces when searching for Integrations and Integration Kits
	*RootCmdOptions
	KitsToDelete []*v1.IntegrationKit
	AssumeYes    bool `mapstructure:"assumeyes"`
	DryRun       bool `mapstructure:"dry-run"`
}

func (o *kitPruneCommandOptions) preRun(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	kits, err := getKits(o.Context, c, o.Namespace)
	if err != nil {
		return err
	}
	integrations, err := getIntegrations(o.Context, c, o.Namespace)
	if err != nil {
		return err
	}
	usedImages := getUsedImages(integrations)
	o.KitsToDelete = make([]*v1.IntegrationKit, 0)
	for _, kit := range kits {
		curr := kit
		if usedImages[kit.Status.Image] == nil {
			o.KitsToDelete = append(o.KitsToDelete, &curr)
		}
	}
	o.printInfo(cmd)
	if o.DryRun || o.AssumeYes || o.nothingToDo() {
		return nil
	}
	o.DryRun = ask(cmd)
	return nil
}

func (o *kitPruneCommandOptions) run(cmd *cobra.Command, args []string) error {
	if o.DryRun || o.nothingToDo() {
		return nil
	}
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	err = deleteKits(o.Context, o.KitsToDelete, c)
	if err != nil {
		return err
	}
	return nil
}

func (o *kitPruneCommandOptions) nothingToDo() bool {
	return len(o.KitsToDelete) == 0
}

func (o *kitPruneCommandOptions) printInfo(cmd *cobra.Command) {
	if o.nothingToDo() {
		fmt.Fprintln(cmd.OutOrStdout(), "Nothing to do")
		return
	}
	if len(o.KitsToDelete) != 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Integration Kits will be deleted:")
	}
	for _, kit := range o.KitsToDelete {
		fmt.Fprintln(cmd.OutOrStdout(), fmt.Sprintf("%s in namespace: %s", kit.Name, kit.Namespace))
	}
	if len(o.KitsToDelete) != 0 {
		fmt.Fprintln(cmd.OutOrStdout(), "\nThe following Images will no longer be used by camel-k and can be deleted from the Image Registry:")
		for _, kit := range o.KitsToDelete {
			fmt.Fprintln(cmd.OutOrStdout(), kit.Status.Image)
		}
	}
}

func deleteKits(context context.Context, toDelete []*v1.IntegrationKit, c client.Client) error {
	for _, kit := range toDelete {
		if err := c.Delete(context, kit); err != nil {
			return err
		}
	}
	return nil
}
