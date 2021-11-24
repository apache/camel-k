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
	"errors"
	"fmt"
	"strconv"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/spf13/cobra"
	k8errors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdDelete --.
func newCmdDelete(rootCmdOptions *RootCmdOptions) (*cobra.Command, *deleteCmdOptions) {
	options := deleteCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "delete [integration1] [integration2] ...",
		Short:   "Delete integrations deployed on Kubernetes",
		PreRunE: decode(&options),
		RunE: func(_ *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			if err := options.run(args); err != nil {
				fmt.Println(err.Error())
			}

			return nil
		},
	}

	cmd.Flags().Bool("all", false, "Delete all integrations")

	return &cmd, &options
}

type deleteCmdOptions struct {
	*RootCmdOptions
	DeleteAll bool `mapstructure:"all"`
}

func (command *deleteCmdOptions) validate(args []string) error {
	if command.DeleteAll && len(args) > 0 {
		return errors.New("invalid combination: both all flag and named integrations are set")
	}
	if !command.DeleteAll && len(args) == 0 {
		return errors.New("invalid combination: neither all flag nor named integrations are set")
	}

	return nil
}

func (command *deleteCmdOptions) run(args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}
	if len(args) != 0 && !command.DeleteAll {
		for _, arg := range args {
			name := kubernetes.SanitizeName(arg)
			integration, err := getIntegration(command.Context, c, name, command.Namespace)
			if err != nil {
				if k8errors.IsNotFound(err) {
					fmt.Println("Integration " + name + " not found. Skipped.")
				} else {
					return err
				}
			} else {
				err := deleteIntegration(command.Context, c, integration)
				if err != nil {
					return err
				}
				fmt.Println("Integration " + name + " deleted")
			}
		}
	} else if command.DeleteAll {
		integrationList := v1.IntegrationList{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKind,
			},
		}

		err := c.List(command.Context, &integrationList, k8sclient.InNamespace(command.Namespace))
		if err != nil {
			return err
		}
		for _, integration := range integrationList.Items {
			integration := integration // pin
			err := deleteIntegration(command.Context, c, &integration)
			if err != nil {
				return err
			}
		}
		if len(integrationList.Items) == 0 {
			fmt.Println("Nothing to delete")
		} else {
			fmt.Println(strconv.Itoa(len(integrationList.Items)) + " integration(s) deleted")
		}
	}

	return nil
}

func getIntegration(ctx context.Context, c client.Client, name string, namespace string) (*v1.Integration, error) {
	key := k8sclient.ObjectKey{
		Name:      name,
		Namespace: namespace,
	}
	answer := v1.NewIntegration(namespace, name)
	if err := c.Get(ctx, key, &answer); err != nil {
		return nil, err
	}
	return &answer, nil
}

func deleteIntegration(ctx context.Context, c client.Client, integration *v1.Integration) error {
	deleted, binding, err := deleteKameletBindingIfExists(ctx, c, integration)
	if err != nil {
		return err
	}
	if deleted {
		// Deleting KameletBinding will automatically clean up the integration
		fmt.Println("KameletBinding " + binding + " deleted")
		return nil
	}
	return c.Delete(ctx, integration)
}

func deleteKameletBindingIfExists(ctx context.Context, c client.Client, integration *v1.Integration) (bool, string, error) {
	kind, name := findCreator(integration)
	if kind != v1alpha1.KameletBindingKind || name == "" {
		return false, "", nil
	}

	binding := v1alpha1.KameletBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       kind,
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: integration.Namespace,
			Name:      name,
		},
	}
	err := c.Delete(ctx, &binding)
	if k8errors.IsNotFound(err) {
		// Simply skip if binding doesn't exist (could be deleted already)
		return false, name, nil
	}
	return err == nil, name, err
}

func findCreator(integration *v1.Integration) (string, string) {
	kind := integration.GetLabels()[kubernetes.CamelCreatorLabelKind]
	name := integration.GetLabels()[kubernetes.CamelCreatorLabelName]
	if kind == "" && name == "" {
		// Look up in OwnerReferences in case creator labels are absent
		for _, owner := range integration.GetOwnerReferences() {
			if owner.Kind == v1alpha1.KameletBindingKind {
				return owner.Kind, owner.Name
			}
		}
	}
	return kind, name
}
