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
	"strings"
	"text/tabwriter"

	"github.com/spf13/cobra"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func newKameletGetCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kameletGetCommandOptions) {
	options := kameletGetCommandOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "get",
		Short:   "Get defined Kamelet",
		Long:    `Get defined Kamelet.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(); err != nil {
				return err
			}
			if err := options.run(cmd); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().Bool("sink", false, "Show only sink Kamelets")
	cmd.Flags().Bool("source", false, "Show only source Kamelets")
	cmd.Flags().Bool("action", false, "Show only action Kamelets")
	cmd.Flags().String("group", "", "Filters Kamelets by group")
	cmd.Flags().Bool("bundled", true, "Includes bundled Kamelets")
	cmd.Flags().Bool("read-only", true, "Includes read-only Kamelets")

	return &cmd, &options
}

type kameletGetCommandOptions struct {
	*RootCmdOptions
	Sink     bool   `mapstructure:"sink"`
	Source   bool   `mapstructure:"source"`
	Action   bool   `mapstructure:"action"`
	Group    string `mapstructure:"group"`
	Bundled  bool   `mapstructure:"bundled"`
	ReadOnly bool   `mapstructure:"read-only"`
}

func (command *kameletGetCommandOptions) validate() error {
	count := 0
	for _, b := range []bool{command.Sink, command.Source, command.Action} {
		if b {
			count++
		}
	}

	if count > 1 {
		return errors.New("invalid combination: flags --sink, --source, and --action are mutually exclusive")
	}
	return nil
}

func (command *kameletGetCommandOptions) run(cmd *cobra.Command) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	klList := v1alpha1.NewKameletList()
	if err := c.List(command.Context, &klList, k8sclient.InNamespace(command.Namespace)); err != nil {
		return err
	}

	w := tabwriter.NewWriter(cmd.OutOrStdout(), 0, 8, 1, '\t', 0)
	fmt.Fprintln(w, "NAME\tPHASE\tTYPE\tGROUP\tBUNDLED\tREAD ONLY\tTITLE")
	for _, kl := range klList.Items {
		klType := kl.Labels[v1alpha1.KameletTypeLabel]
		group := kl.Annotations[v1alpha1.KameletGroupLabel]
		bundled := kl.Labels[v1alpha1.KameletBundledLabel]
		readOnly := kl.Labels[v1alpha1.KameletReadOnlyLabel]

		if command.Sink && klType != v1alpha1.KameletTypeSink {
			continue
		}
		if command.Source && klType != v1alpha1.KameletTypeSource {
			continue
		}
		if command.Action && klType != v1alpha1.KameletTypeAction {
			continue
		}
		if command.Group != "" && !strings.EqualFold(command.Group, group) {
			continue
		}
		if !command.Bundled && bundled == "true" {
			continue
		}
		if !command.ReadOnly && readOnly == "true" {
			continue
		}

		fmt.Fprintf(w, "%s\t%s\t%s\t%s\t%s\t%s\t%s\n",
			kl.Name,
			string(kl.Status.Phase),
			klType,
			group,
			bundled,
			readOnly,
			kl.Spec.Definition.Title)
	}

	return w.Flush()
}
