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

	"github.com/apache/camel-k/pkg/install"
	"github.com/spf13/cobra"
	"k8s.io/apimachinery/pkg/api/errors"
	"os"
)

type InstallCmdOptions struct {
	*RootCmdOptions
	ClusterSetupOnly bool
}

func NewCmdInstall(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := InstallCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "install",
		Short: "Install Camel K on a Kubernetes cluster",
		Long:  `Installs Camel K on a Kubernetes or Openshift cluster.`,
		RunE:  options.install,
	}

	cmd.Flags().BoolVar(&options.ClusterSetupOnly, "cluster-setup", false, "Execute cluster-wide operations only (may require admin rights)")
	cmd.ParseFlags(os.Args)

	return &cmd
}

func (o *InstallCmdOptions) install(cmd *cobra.Command, args []string) error {
	err := install.SetupClusterwideResources()
	if err != nil && errors.IsForbidden(err) {
		// TODO explain that this is a one time operation and add a flag to do cluster-level operations only when logged as admin
		fmt.Println("Current user is not authorized to create cluster-wide objects like custom resource definitions or cluster roles: ", err)
		fmt.Println("Please login as cluster-admin and execute \"kamel install --cluster-setup\" to install those resources (one-time operation).")
		return nil // TODO better error handling: if here we return err the help page is shown
	}

	if o.ClusterSetupOnly {
		fmt.Println("Camel K cluster setup completed successfully")
	} else {
		namespace := o.Namespace

		err = install.InstallOperator(namespace)
		if err != nil {
			return err
		}
		fmt.Println("Camel K installed in namespace", namespace)
	}

	return nil
}
