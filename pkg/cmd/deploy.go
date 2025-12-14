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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/spf13/cobra"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// newCmdDeploy --.
func newCmdDeploy(rootCmdOptions *RootCmdOptions) (*cobra.Command, *deployCmdOptions) {
	options := deployCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:     "deploy <name>",
		Short:   "Deploy an Integration or Pipe that was previously built with --dont-run-after-build flag",
		PreRunE: decode(&options, options.Flags),
		RunE:    options.run,
	}

	return &cmd, &options
}

type deployCmdOptions struct {
	*RootCmdOptions
}

func (o *deployCmdOptions) validate(_ *cobra.Command, args []string) error {
	if len(args) != 1 {
		return errors.New("deploy requires an Integration or Pipe name argument")
	}

	return nil
}

func (o *deployCmdOptions) run(cmd *cobra.Command, args []string) error {
	if err := o.validate(cmd, args); err != nil {
		return err
	}

	name := args[0]
	c, err := o.GetCmdClient()
	if err != nil {
		return fmt.Errorf("could not retrieve cluster client: %w", err)
	}

	existing, err := getIntegration(o.Context, c, name, o.Namespace)
	if err != nil {
		return fmt.Errorf("could not get Integration or Pipe "+name+": %w", err)
	}
	if existing.Status.Phase != v1.IntegrationPhaseBuildComplete {
		return fmt.Errorf("could not run an Integration in %s status", existing.Status.Phase)
	}

	integration := existing.DeepCopy()
	integration.Status.Phase = v1.IntegrationPhaseDeploying

	patch := ctrl.MergeFrom(existing)
	d, err := patch.Data(integration)
	if err != nil {
		return err
	}

	if string(d) == "{}" {
		fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" unchanged`)

		return nil
	}
	err = c.Status().Patch(o.Context, integration, patch)
	if err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), `Integration "`+name+`" deployed`)

	return nil
}
