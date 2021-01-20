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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/spf13/cobra"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
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

func displayOperatorVersion(ctx context.Context, c client.Client, namespace string) {
	operatorVersion, err := operatorVersion(ctx, c, namespace)
	if err == nil {
		fmt.Printf("Camel K Operator %s\n", operatorVersion)
	} else {
		fmt.Printf("Error while looking for camel-k operator in namespace %s (%s)\n", namespace, err)
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
	v1, err := semver.NewVersion(aVersion)
	if err != nil {
		fmt.Printf("Could not parse %s (error: %s)\n", v1, err)
		return false
	}
	v2, err := semver.NewVersion(bVersion)
	if err != nil {
		fmt.Printf("Could not parse %s (error: %s)\n", v2, err)
		return false
	}
	// We consider compatible when major and minor are equals
	return v1.Major() == v2.Major() && v1.Minor() == v2.Minor()
}
