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
	"github.com/Masterminds/semver"
	"github.com/spf13/cobra"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
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
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}
		displayOperatorVersion(cmd, o.Context, c, o.Namespace)
	} else {
		displayClientVersion(cmd)
	}
	return nil
}

func displayClientVersion(cmd *cobra.Command) {
	if VersionVariant != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Client %s %s\n", VersionVariant, defaults.Version)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Client %s\n", defaults.Version)
	}
}

func displayOperatorVersion(cmd *cobra.Command, ctx context.Context, c client.Client, namespace string) {
	operatorVersion, err := operatorVersion(ctx, c, namespace)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Unable to retrieve operator version: %s\n", err)
	} else {
		if operatorVersion == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Unable to retrieve operator version: The IntegrationPlatform resource hasn't been reconciled yet!")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Camel K Operator %s\n", operatorVersion)
		}
	}
}

func operatorVersion(ctx context.Context, c client.Client, namespace string) (string, error) {
	platform := v1.NewIntegrationPlatform(namespace, "camel-k")
	platformKey := k8sclient.ObjectKey{
		Namespace: namespace,
		Name:      "camel-k",
	}

	if err := c.Get(ctx, platformKey, &platform); err == nil {
		return platform.Status.Version, nil
	} else {
		return "", err
	}
}

func compatibleVersions(aVersion, bVersion string) bool {
	a, err := semver.NewVersion(aVersion)
	if err != nil {
		fmt.Printf("Could not parse %s (error: %s)\n", a, err)
		return false
	}
	b, err := semver.NewVersion(bVersion)
	if err != nil {
		fmt.Printf("Could not parse %s (error: %s)\n", b, err)
		return false
	}
	// We consider compatible when major and minor are equals
	return a.Major() == b.Major() && a.Minor() == b.Minor()
}
