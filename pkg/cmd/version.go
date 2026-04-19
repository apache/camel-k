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

	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// VersionVariant may be overridden at build time.
var VersionVariant = ""

const (
	infoVersion = "Version"
)

func newCmdVersion(rootCmdOptions *RootCmdOptions) (*cobra.Command, *versionCmdOptions) {
	options := versionCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:               "version",
		Short:             "Display client version",
		Long:              `Display Camel K client version.`,
		PersistentPreRunE: decode(&options, options.Flags),
		PreRunE:           options.preRunE,
		RunE:              options.run,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().Bool("operator", false, "Display Operator version. Deprecated option.")
	cmd.Flags().BoolP("verbose", "v", false, "Display all available extra information")
	cmd.Flags().BoolP("all", "a", false, "Display both Client and Operator version. Deprecated option.")

	return &cmd, &options
}

type versionCmdOptions struct {
	*RootCmdOptions

	// Deprecated: to be removed in future versions.
	Operator bool `mapstructure:"operator"`
	Verbose  bool `mapstructure:"verbose"`
	// Deprecated: to be removed in future versions.
	All bool `mapstructure:"all"`
}

func (o *versionCmdOptions) preRunE(cmd *cobra.Command, args []string) error {
	if !o.Operator && !o.All {
		// let the command to work in offline mode
		cmd.Annotations[offlineCommandLabel] = "true"
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Operator version discovery is deprecated. It will be removed from future releases.\n")
	}

	return o.preRun(cmd, args)
}

func (o *versionCmdOptions) run(cmd *cobra.Command, _ []string) error {
	if o.All || !o.Operator {
		o.displayClientVersion(cmd)
	}
	if o.All {
		// breaking line
		fmt.Fprintln(cmd.OutOrStdout(), "")
	}
	if o.All || o.Operator {
		c, err := o.GetCmdClient()
		if err != nil {
			return err
		}
		o.displayOperatorVersion(cmd, c)
	}

	return nil
}

func (o *versionCmdOptions) displayClientVersion(cmd *cobra.Command) {
	if VersionVariant != "" {
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Client %s %s\n", VersionVariant, defaults.Version)
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "Camel K Client %s\n", defaults.Version)
	}
	if o.Verbose {
		fmt.Fprintf(cmd.OutOrStdout(), "Git Commit: %s\n", defaults.GitCommit)
	}
}

func (o *versionCmdOptions) displayOperatorVersion(cmd *cobra.Command, c client.Client) {
	operatorInfo, err := operatorInfo(o.Context, c, o.Namespace)
	if err != nil {
		fmt.Fprintf(cmd.OutOrStdout(), "Unable to retrieve operator version: %s\n", err)
	} else {
		if operatorInfo[infoVersion] == "" {
			fmt.Fprintf(cmd.OutOrStdout(), "Unable to retrieve operator version: The IntegrationPlatform resource hasn't been reconciled yet!")
		} else {
			fmt.Fprintf(cmd.OutOrStdout(), "Camel K Operator %s\n", operatorInfo[infoVersion])
			if o.Verbose {
				for k, v := range operatorInfo {
					if k != infoVersion {
						fmt.Fprintf(cmd.OutOrStdout(), "%s: %s\n", k, v)
					}
				}
			}
		}
	}
}

// Deprecated: to be removed in future versions.
func operatorInfo(ctx context.Context, c client.Client, namespace string) (map[string]string, error) {
	infos := make(map[string]string)

	pl, err := platform.GetOrFindLocal(ctx, c, namespace)
	if err != nil && k8serrors.IsNotFound(err) {
		// find default operator platform in any namespace
		if defaultPlatform, _ := platform.LookupForPlatformName(ctx, c, platform.DefaultPlatformName); defaultPlatform != nil {
			pl = defaultPlatform
		} else {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	infos["Name"] = pl.Name
	infos["Version"] = pl.Status.Version
	infos["Publish strategy"] = string(pl.Status.Build.PublishStrategy)
	infos["Runtime version"] = pl.Status.Build.RuntimeVersion
	infos["Registry address"] = pl.Status.Build.Registry.Address
	infos["Git commit"] = pl.Status.Info["gitCommit"]

	catalog, err := camel.LoadCatalog(ctx, c, pl.Namespace, v1.RuntimeSpec{Version: pl.Status.Build.RuntimeVersion, Provider: pl.Status.Build.RuntimeProvider})
	if err != nil {
		return nil, err
	}
	if catalog != nil {
		infos["Camel Quarkus version"] = catalog.GetCamelQuarkusVersion()
		infos["Camel version"] = catalog.GetCamelVersion()
		infos["Quarkus version"] = catalog.GetQuarkusVersion()
	}

	return infos, nil
}
