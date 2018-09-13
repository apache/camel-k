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
	"strconv"

	"github.com/operator-framework/operator-sdk/pkg/sdk"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

// NewCmdContext --
func newContextCreateCmd(rootCmdOptions *RootCmdOptions) *cobra.Command {
	impl := &contextCreateCommand{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:   "create",
		Short: "Create an Integration Context",
		Long:  `Create an Integration Context.`,
		Args:  impl.validateArgs,
		RunE:  impl.run,
	}

	cmd.Flags().StringSliceVarP(&impl.dependencies, "dependency", "d", nil, "Add a dependency")
	cmd.Flags().StringSliceVarP(&impl.properties, "property", "p", nil, "Add a camel property")
	cmd.Flags().StringSliceVar(&impl.configmaps, "configmap", nil, "Add a ConfigMap")
	cmd.Flags().StringSliceVar(&impl.secrets, "secret", nil, "Add a Secret")

	return &cmd
}

type contextCreateCommand struct {
	*RootCmdOptions

	dependencies []string
	properties   []string
	configmaps   []string
	secrets      []string
}

func (command *contextCreateCommand) validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("accepts 1 arg, received " + strconv.Itoa(len(args)))
	}

	return nil
}

func (command *contextCreateCommand) run(cmd *cobra.Command, args []string) error {
	namespace := command.Namespace
	name := kubernetes.SanitizeName(args[0])

	ctx := v1alpha1.NewIntegrationContext(namespace, name)
	ctx.Spec = v1alpha1.IntegrationContextSpec{
		Dependencies:  command.dependencies,
		Configuration: make([]v1alpha1.ConfigurationSpec, 0),
	}

	for _, item := range command.properties {
		ctx.Spec.Configuration = append(ctx.Spec.Configuration, v1alpha1.ConfigurationSpec{
			Type:  "property",
			Value: item,
		})
	}
	for _, item := range command.configmaps {
		ctx.Spec.Configuration = append(ctx.Spec.Configuration, v1alpha1.ConfigurationSpec{
			Type:  "configmap",
			Value: item,
		})
	}
	for _, item := range command.secrets {
		ctx.Spec.Configuration = append(ctx.Spec.Configuration, v1alpha1.ConfigurationSpec{
			Type:  "secret",
			Value: item,
		})
	}

	existed := false
	err := sdk.Create(&ctx)
	if err != nil && k8serrors.IsAlreadyExists(err) {
		existed = true
		clone := ctx.DeepCopy()
		err = sdk.Get(clone)
		if err != nil {
			return err
		}
		ctx.ResourceVersion = clone.ResourceVersion
		err = sdk.Update(&ctx)
	}

	if err != nil {
		return err
	}

	if !existed {
		fmt.Printf("integration context \"%s\" created\n", name)
	} else {
		fmt.Printf("integration context \"%s\" updated\n", name)
	}

	return nil
}
