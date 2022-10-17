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
	"regexp"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	platformutil "github.com/apache/camel-k/pkg/platform"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// kameletRepositoryURIRegexp is the regular expression used to validate the URI of a Kamelet repository.
var kameletRepositoryURIRegexp = regexp.MustCompile(`^github:[^/]+/[^/]+((/[^/]+)*)?$`)

func newKameletAddRepoCmd(rootCmdOptions *RootCmdOptions) (*cobra.Command, *kameletAddRepoCommandOptions) {
	options := kameletAddRepoCommandOptions{
		kameletUpdateRepoCommandOptions: &kameletUpdateRepoCommandOptions{
			RootCmdOptions: rootCmdOptions,
		},
	}

	cmd := cobra.Command{
		Use:     "add-repo github:owner/repo[/path_to_kamelets_folder][@version] ...",
		Short:   "Add a Kamelet repository",
		Long:    `Add a Kamelet repository.`,
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

type kameletUpdateRepoCommandOptions struct {
	*RootCmdOptions
	OperatorID string `mapstructure:"operator-id" yaml:",omitempty"`
}

type kameletAddRepoCommandOptions struct {
	*kameletUpdateRepoCommandOptions
}

func (o *kameletAddRepoCommandOptions) validate(args []string) error {
	if len(args) == 0 {
		return errors.New("at least one Kamelet repository is expected")
	}
	return nil
}

func (o *kameletAddRepoCommandOptions) run(cmd *cobra.Command, args []string) error {
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
		if err := checkURI(uri, platform.Spec.Kamelet.Repositories); err != nil {
			return err
		}
		platform.Spec.Kamelet.Repositories = append(platform.Spec.Kamelet.Repositories, v1.IntegrationPlatformKameletRepositorySpec{
			URI: uri,
		})
	}
	return c.Update(o.Context, platform)
}

// getIntegrationPlatform gives the integration platform matching with the operator id in the provided namespace.
func (o *kameletUpdateRepoCommandOptions) getIntegrationPlatform(cmd *cobra.Command, c client.Client) (*v1.IntegrationPlatform, error) {
	key := client.ObjectKey{
		Namespace: o.Namespace,
		Name:      o.OperatorID,
	}
	platform := v1.IntegrationPlatform{}
	if err := c.Get(o.Context, key, &platform); err != nil {
		if k8serrors.IsNotFound(err) {
			// IntegrationPlatform may be in the operator namespace, but we currently don't have a way to determine it: we just warn
			fmt.Fprintf(cmd.ErrOrStderr(), "Warning: IntegrationPlatform %q not found in namespace %q\n", key.Name, key.Namespace)
			return nil, nil
		}
		return nil, err
	}
	return &platform, nil
}

// findIntegrationPlatform gives the primary integration platform that could be found in the provided namespace.
func (o *kameletUpdateRepoCommandOptions) findIntegrationPlatform(cmd *cobra.Command, c client.Client) (*v1.IntegrationPlatform, error) {
	platforms, err := platformutil.ListPrimaryPlatforms(o.Context, c, o.Namespace)
	if err != nil {
		return nil, err
	}
	for _, p := range platforms.Items {
		p := p // pin
		if platformutil.IsActive(&p) {
			return &p, nil
		}
	}
	fmt.Fprintf(cmd.ErrOrStderr(), "Warning: No active primary IntegrationPlatform could be found in namespace %q\n", o.Namespace)
	return nil, nil
}

func checkURI(uri string, repositories []v1.IntegrationPlatformKameletRepositorySpec) error {
	if !kameletRepositoryURIRegexp.MatchString(uri) {
		return fmt.Errorf("malformed Kamelet repository uri %s, the expected format is github:owner/repo[/path_to_kamelets_folder][@version]", uri)
	}
	for _, repo := range repositories {
		if repo.URI == uri {
			return fmt.Errorf("duplicate Kamelet repository uri %s", uri)
		}
	}
	return nil
}
