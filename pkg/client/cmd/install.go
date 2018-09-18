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
	"fmt"

	"os"

	"github.com/apache/camel-k/pkg/install"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"github.com/pkg/errors"
)

// NewCmdInstall --
func NewCmdInstall(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := installCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "install",
		Short: "Install Camel K on a Kubernetes cluster",
		Long:  `Installs Camel K on a Kubernetes or Openshift cluster.`,
		RunE:  options.install,
	}

	cmd.Flags().BoolVar(&options.clusterSetupOnly, "cluster-setup", false, "Execute cluster-wide operations only (may require admin rights)")
	cmd.ParseFlags(os.Args)

	return &cmd
}

type installCmdOptions struct {
	*RootCmdOptions
	clusterSetupOnly bool
}

func (o *installCmdOptions) install(cmd *cobra.Command, args []string) error {
	err := install.SetupClusterwideResources()
	if err != nil && k8serrors.IsForbidden(err) {
		fmt.Println("Current user is not authorized to create cluster-wide objects like custom resource definitions or cluster roles: ", err)
		return errors.New("please login as cluster-admin and execute \"kamel install --cluster-setup\" to install cluster-wide resources (one-time operation)")
	}

	if o.clusterSetupOnly {
		fmt.Println("Camel K cluster setup completed successfully")
	} else {
		namespace := o.Namespace

		err = install.Operator(namespace)
		if err != nil {
			return err
		}

		err = install.PlatformContexts(namespace)
		if err != nil {
			return err
		}

		fmt.Println("Camel K installed in namespace", namespace)
	}

	return nil
}
