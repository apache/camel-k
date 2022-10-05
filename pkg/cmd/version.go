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
	"strings"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/Masterminds/semver"
	"github.com/fatih/camelcase"
	"github.com/spf13/cobra"

	"github.com/apache/camel-k/pkg/client"
	platformutil "github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/log"
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
		PersistentPreRunE: decode(&options),
		PreRunE:           options.preRunE,
		RunE:              options.run,
		Annotations:       make(map[string]string),
	}

	cmd.Flags().Bool("operator", false, "Display Operator version")
	cmd.Flags().BoolP("verbose", "v", false, "Display all available extra information")
	cmd.Flags().BoolP("all", "a", false, "Display both Client and Operator version")

	return &cmd, &options
}

type versionCmdOptions struct {
	*RootCmdOptions
	Operator bool `mapstructure:"operator"`
	Verbose  bool `mapstructure:"verbose"`
	All      bool `mapstructure:"all"`
}

func (o *versionCmdOptions) preRunE(cmd *cobra.Command, args []string) error {
	if !o.Operator && !o.All {
		// let the command to work in offline mode
		cmd.Annotations[offlineCommandLabel] = "true"
	}
	return o.RootCmdOptions.preRun(cmd, args)
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

func operatorInfo(ctx context.Context, c client.Client, namespace string) (map[string]string, error) {
	infos := make(map[string]string)

	platform, err := platformutil.GetOrFindLocal(ctx, c, namespace)
	if err != nil && k8serrors.IsNotFound(err) {
		// find default operator platform in any namespace
		if defaultPlatform, _ := platformutil.LookupForPlatformName(ctx, c, platformutil.DefaultPlatformName); defaultPlatform != nil {
			platform = defaultPlatform
		} else {
			return nil, err
		}
	} else if err != nil {
		return nil, err
	}

	// Useful information
	infos["name"] = platform.Name
	infos["version"] = platform.Status.Version
	infos["publishStrategy"] = string(platform.Status.Build.PublishStrategy)
	infos["runtimeVersion"] = platform.Status.Build.RuntimeVersion
	infos["registryAddress"] = platform.Status.Build.Registry.Address

	if platform.Status.Info != nil {
		for k, v := range platform.Status.Info {
			infos[k] = v
		}
	}

	ccInfo := fromCamelCase(infos)
	log.Debugf("Operator Info for namespace %s: %v", namespace, ccInfo)
	return ccInfo, nil
}

func fromCamelCase(infos map[string]string) map[string]string {
	textKeys := make(map[string]string)
	for k, v := range infos {
		key := strings.Title(strings.Join(camelcase.Split(k), " "))
		textKeys[key] = v
	}

	return textKeys
}

func operatorVersion(ctx context.Context, c client.Client, namespace string) (string, error) {
	infos, err := operatorInfo(ctx, c, namespace)
	if err != nil {
		return "", err
	}
	return infos[infoVersion], nil
}

func compatibleVersions(aVersion, bVersion string, cmd *cobra.Command) bool {
	a, err := semver.NewVersion(aVersion)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Could not parse '"+aVersion+"' (error:", err.Error()+")")
		return false
	}
	b, err := semver.NewVersion(bVersion)
	if err != nil {
		fmt.Fprintln(cmd.ErrOrStderr(), "Could not parse '"+bVersion+"' (error:", err.Error()+")")
		return false
	}
	// We consider compatible when major and minor are equals
	return a.Major() == b.Major() && a.Minor() == b.Minor()
}
