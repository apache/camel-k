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
	k8errors "k8s.io/apimachinery/pkg/api/errors"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func newKameletDeleteCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kameletDeleteCommandOptions) {
	options := kameletDeleteCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "delete [Kamelet1] [Kamelet2] ...",
		Short:   "Delete Kamelets deployed on Kubernetes",
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			return options.run(cmd, args)
		},
	}

	cmd.Flags().Bool("all", false, "Delete all Kamelets")

	return &cmd, &options
}

type kameletDeleteCommandOptions struct {
	*RootCmdOptions
	All bool `mapstructure:"all"`
}

func (command *kameletDeleteCommandOptions) validate(args []string) error {
	if command.All && len(args) > 0 {
		return errors.New("invalid combination: --all flag is set and at least one kamelet name is provided")
	}
	if !command.All && len(args) == 0 {
		return errors.New("invalid combination: provide one or several kamelet names or set --all flag for all kamelets")
	}

	return nil
}

func (command *kameletDeleteCommandOptions) run(cmd *cobra.Command, args []string) error {
	names := args

	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	if command.All {
		klList := v1alpha1.NewKameletList()
		if err := c.List(command.Context, &klList, k8sclient.InNamespace(command.Namespace)); err != nil {
			return err
		}
		names = make([]string, 0, len(klList.Items))
		for _, kl := range klList.Items {
			// only include non-bundled, non-readonly kamelets
			if kl.Labels[v1alpha1.KameletBundledLabel] != "true" && kl.Labels[v1alpha1.KameletReadOnlyLabel] != "true" {
				names = append(names, kl.Name)
			}
		}
	}

	for _, name := range names {
		if err := command.delete(cmd, name); err != nil {
			return err
		}
	}

	return nil
}

func (command *kameletDeleteCommandOptions) delete(cmd *cobra.Command, name string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	kl := v1alpha1.NewKamelet(command.Namespace, name)
	key := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      name,
	}
	err = c.Get(command.Context, key, &kl)
	if err != nil {
		if k8errors.IsNotFound(err) {
			return fmt.Errorf("no kamelet found with name \"%s\"", name)
		}
		return err
	}

	// check that it is not a bundled nor read-only one which is supposed to belong to platform
	// thus not managed by the end user
	if kl.Labels[v1alpha1.KameletBundledLabel] == "true" || kl.Labels[v1alpha1.KameletReadOnlyLabel] == "true" {
		// skip platform Kamelets while deleting all Kamelets
		if command.All {
			return nil
		}
		return fmt.Errorf("kamelet \"%s\" is not editable", name)
	}

	err = c.Delete(command.Context, &kl)
	if err != nil {
		if k8errors.IsNotFound(err) {
			return fmt.Errorf("no kamelet found with name \"%s\"", name)
		}
		return fmt.Errorf("error deleting kamelet \"%s\": %w", name, err)
	}
	fmt.Fprintln(cmd.OutOrStdout(), `kamelet "`+name+`" has been deleted`)
	return nil
}
