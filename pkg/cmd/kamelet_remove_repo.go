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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/spf13/cobra"
)

func newKameletRemoveRepoCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kameletRemoveRepoCommandOptions) {
	options := kameletRemoveRepoCommandOptions{
		kameletUpdateRepoCommandOptions: &kameletUpdateRepoCommandOptions{
			RootCmdOptions: rootCmdOptions,
		},
	}

	cmd := cobra.Command{
		Use:     "remove-repo github:owner/repo[/path_to_kamelets_folder][@version] ...",
		Short:   "Remove a Kamelet repository",
		Long:    `Remove a Kamelet repository.`,
		PreRunE: decode(&options),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := options.validate(args); err != nil {
				return err
			}
			return options.run(cmd, args)
		},
	}

	cmd.Flags().StringP("operator-id", "x", "", "Id of the Operator to update. If not set, the active primary Integration Platform is updated.")

	return &cmd, &options
}

type kameletRemoveRepoCommandOptions struct {
	*kameletUpdateRepoCommandOptions
}

func (o *kameletRemoveRepoCommandOptions) validate(args []string) error {
	if len(args) == 0 {
		return errors.New("at least one Kamelet repository is expected")
	}
	return nil
}

func (o *kameletRemoveRepoCommandOptions) run(cmd *cobra.Command, args []string) error {
	c, err := o.GetCmdClient()
	if err != nil {
		return err
	}
	var platform *v1.IntegrationPlatform
	if o.OperatorID == "" {
		platform, err = o.findIntegrationPlatform(cmd, c)
	} else {
		platform, err = o.getIntegrationPlatform(cmd, c)
	}
	if err != nil {
		return err
	} else if platform == nil {
		return nil
	}
	for _, uri := range args {
		i, err := getURIIndex(uri, platform.Spec.Kamelet.Repositories)
		if err != nil {
			return err
		}
		platform.Spec.Kamelet.Repositories[i] = platform.Spec.Kamelet.Repositories[len(platform.Spec.Kamelet.Repositories)-1]
		platform.Spec.Kamelet.Repositories = platform.Spec.Kamelet.Repositories[:len(platform.Spec.Kamelet.Repositories)-1]
	}
	return c.Update(o.Context, platform)
}

func getURIIndex(uri string, repositories []v1.IntegrationPlatformKameletRepositorySpec) (int, error) {
	for i, repo := range repositories {
		if repo.URI == uri {
			return i, nil
		}
	}
	return 0, fmt.Errorf("non existing Kamelet repository uri %s", uri)
}
