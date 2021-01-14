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
	"regexp"

	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/pkg/errors"
	v1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/spf13/cobra"
)

// VersionVariant may be overridden at build time
var VersionVariant = ""

func newCmdVersion(rootCmdOptions *RootCmdOptions) (*cobra.Command, *versionCmdOptions) {
	options := versionCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:               "version",
		Short:             "Display client version",
		Long:              `Display Camel K client version.`,
		PersistentPreRunE: decode(&options),
		PreRunE:           options.preRunE,
		RunE:              options.run,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().Bool("operator", false, "Display Operator version")

	return &cmd, &options
}

type versionCmdOptions struct {
	*RootCmdOptions
	Operator bool `mapstructure:"operator"`
}

func (o *versionCmdOptions) preRunE(cmd *cobra.Command, args []string) error {
	if !o.Operator {
		// let the command to work in offline mode
		cmd.Annotations[offlineCommandLabel] = "true"
	}
	return o.RootCmdOptions.preRun(cmd, args)
}

func (o *versionCmdOptions) run(cmd *cobra.Command, _ []string) error {
	if o.Operator {
		client, err := o.GetCmdClient()
		if err != nil {
			return err
		}
		displayOperatorVersion(o.Context, client, o.Namespace)
	} else {
		displayClientVersion()
	}
	return nil
}

func displayClientVersion() {
	if VersionVariant != "" {
		fmt.Printf("Camel K Client %s %s\n", VersionVariant, defaults.Version)
	} else {
		fmt.Printf("Camel K Client %s\n", defaults.Version)
	}
}

func displayOperatorVersion(ctx context.Context, client client.Client, namespace string) {
	operatorVersion, err := operatorVersion(ctx, client, namespace)
	if err != nil {
		fmt.Printf("Some issue happened while looking for camel-k operator in namespace %s (error: %s)\n", namespace, err)
	} else {
		fmt.Printf("Camel K Operator %s\n", operatorVersion)
	}
}

func operatorVersion(ctx context.Context, c client.Client, namespace string) (string, error) {
	deployment := v1.Deployment{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Deployment",
			APIVersion: "apps/v1",
		},
	}
	key := types.NamespacedName{Namespace: namespace, Name: "camel-k-operator"}
	err := c.Get(ctx, key, &deployment)
	if err != nil {
		return "", err
	}
	return extractVersionFromDockerImage(deployment.Spec.Template.Spec.Containers[0].Image)
}

func extractVersionFromDockerImage(in string) (string, error) {
	re := regexp.MustCompile(`docker.io/apache/camel-k:(.*?)$`)
	match := re.FindStringSubmatch(in)
	if len(match) == 2 {
		return match[1], nil
	}
	return "", errors.New("Something wrong happened while parsing camel k operator image " + in)
}
