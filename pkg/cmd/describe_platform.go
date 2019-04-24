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
	"io"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/spf13/cobra"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func newDescribePlatformCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {

	impl := &describePlatformCommand{
		rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "platform",
		Short: "Describe an Integration Platform",
		Long:  `Describe an Integration Platform.`,
		RunE: func(_ *cobra.Command, args []string) error {
			if err := impl.validate(args); err != nil {
				return err
			}
			if err := impl.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	return &cmd
}

type describePlatformCommand struct {
	*RootCmdOptions
}

func (command *describePlatformCommand) validate(args []string) error {
	if len(args) != 1 {
		return fmt.Errorf("accepts at least 1 arg, received %d", len(args))
	}
	return nil
}

func (command *describePlatformCommand) run(args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	ctx := v1alpha1.NewIntegrationPlatform(command.Namespace, args[0])
	key := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}

	if err := c.Get(command.Context, key, &ctx); err == nil {
		fmt.Print(command.describeIntegrationPlatform(ctx))
	} else {
		fmt.Printf("IntegrationPlatform '%s' does not exist.\n", args[0])
	}

	return nil
}

func (command *describePlatformCommand) describeIntegrationPlatform(i v1alpha1.IntegrationPlatform) string {
	return indentedString(func(out io.Writer) {
		w := newIndentedWriter(out)
		describeObjectMeta(w, i.ObjectMeta)
		w.write(0, "Phase:\t%s\n", i.Status.Phase)
		w.write(0, "Base Image:\t%s\n", i.Spec.Build.BaseImage)
		w.write(0, "Camel Version:\t%s\n", i.Spec.Build.CamelVersion)
		w.write(0, "Local Repository:\t%s\n", i.Spec.Build.LocalRepository)
		w.write(0, "Publish Strategy:\t%s\n", i.Spec.Build.PublishStrategy)

		if len(i.Spec.Resources.Contexts) > 0 {
			w.write(0, "Resources:\n")
			w.write(1, "Contexts:\n")
			for _, context := range i.Spec.Resources.Contexts {
				w.write(2, "%s\n", context)
			}
		}
	})
}
