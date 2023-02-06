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
	"io"

	"github.com/spf13/cobra"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/indentedwriter"
)

func newDescribePlatformCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *describePlatformCommandOptions) {
	options := describePlatformCommandOptions{
		rootCmdOptions,
	}

	cmd := cobra.Command{
		Use:     "platform",
		Aliases: []string{"ip"},
		Short:   "Describe an Integration Platform",
		Long:    `Describe an Integration Platform.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(cmd, args); err != nil {
				return err
			}
			if err := options.run(cmd, args); err != nil {
				fmt.Fprintln(cmd.OutOrStderr(), err.Error())
			}

			return nil
		},
	}

	return &cmd, &options
}

type describePlatformCommandOptions struct {
	*RootCmdOptions
}

func (command *describePlatformCommandOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("describe expects a platform name argument")
	}
	return nil
}

func (command *describePlatformCommandOptions) run(cmd *cobra.Command, args []string) error {
	c, err := command.GetCmdClient()
	if err != nil {
		return err
	}

	platform := v1.NewIntegrationPlatform(command.Namespace, args[0])
	platformKey := k8sclient.ObjectKey{
		Namespace: command.Namespace,
		Name:      args[0],
	}

	if err := c.Get(command.Context, platformKey, &platform); err == nil {
		if desc, err := command.describeIntegrationPlatform(cmd, platform); err == nil {
			fmt.Fprint(cmd.OutOrStdout(), desc)
		} else {
			fmt.Fprintln(cmd.ErrOrStderr(), err)
		}
	} else {
		fmt.Fprintf(cmd.OutOrStdout(), "IntegrationPlatform '%s' does not exist.\n", args[0])
	}

	return nil
}

func (command *describePlatformCommandOptions) describeIntegrationPlatform(cmd *cobra.Command, platform v1.IntegrationPlatform) (string, error) {
	return indentedwriter.IndentedString(func(out io.Writer) error {
		w := indentedwriter.NewWriter(cmd.OutOrStdout())
		describeObjectMeta(w, platform.ObjectMeta)
		w.Writef(0, "Phase:\t%s\n", platform.Status.Phase)
		w.Writef(0, "Version:\t%s\n", platform.Status.Version)
		w.Writef(0, "Base Image:\t%s\n", platform.GetActualValue(getPlatformBaseImage))
		w.Writef(0, "Runtime Version:\t%s\n", platform.GetActualValue(getPlatformRuntimeVersion))
		w.Writef(0, "Local Repository:\t%s\n", platform.GetActualValue(getPlatformMavenLocalRepository))
		w.Writef(0, "Publish Strategy:\t%s\n", platform.GetActualValue(getPlatformPublishStrategy))

		return nil
	})
}

func getPlatformBaseImage(spec v1.IntegrationPlatformSpec) string {
	return spec.Build.BaseImage
}

func getPlatformRuntimeVersion(spec v1.IntegrationPlatformSpec) string {
	return spec.Build.RuntimeVersion
}

func getPlatformMavenLocalRepository(spec v1.IntegrationPlatformSpec) string {
	return spec.Build.Maven.LocalRepository
}

func getPlatformPublishStrategy(spec v1.IntegrationPlatformSpec) string {
	return string(spec.Build.PublishStrategy)
}
