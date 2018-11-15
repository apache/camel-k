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
	"github.com/apache/camel-k/pkg/util/kubernetes"

	"os"
	"time"

	"github.com/apache/camel-k/pkg/install"
	"github.com/operator-framework/operator-sdk/pkg/k8sclient"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

func newCmdInstall(rootCmdOptions *RootCmdOptions) *cobra.Command {
	options := installCmdOptions{
		RootCmdOptions: rootCmdOptions,
	}
	cmd := cobra.Command{
		Use:   "install",
		Short: "Install Camel K on a Kubernetes cluster",
		Long:  `Installs Camel K on a Kubernetes or OpenShift cluster.`,
		RunE:  options.install,
	}

	cmd.Flags().BoolVar(&options.clusterSetupOnly, "cluster-setup", false, "Execute cluster-wide operations only (may require admin rights)")
	cmd.Flags().BoolVar(&options.exampleSetup, "example", false, "Install example integration")
	cmd.Flags().StringVar(&options.registry, "registry", "", "A Docker registry that can be used to publish images")
	cmd.Flags().StringVarP(&options.outputFormat, "output", "o", "", "Output format. One of: json|yaml")
	cmd.Flags().StringVar(&options.organization, "organization", "", "A organization on the Docker registry that can be used to publish images")
	cmd.Flags().StringVar(&options.pushSecret, "push-secret", "", "A secret used to push images to the Docker registry")
	cmd.ParseFlags(os.Args)

	return &cmd
}

type installCmdOptions struct {
	*RootCmdOptions
	clusterSetupOnly bool
	exampleSetup     bool
	registry         string
	outputFormat     string
	organization     string
	pushSecret       string
}

func (o *installCmdOptions) install(cmd *cobra.Command, args []string) error {
	// Let's use a fast refresh period when running with the CLI
	k8sclient.ResetCacheEvery(8 * time.Second)

	var collection *kubernetes.Collection
	if o.outputFormat != "" {
		collection = kubernetes.NewCollection()
	}

	err := install.SetupClusterwideResourcesOrCollect(collection)
	if err != nil && k8serrors.IsForbidden(err) {
		fmt.Println("Current user is not authorized to create cluster-wide objects like custom resource definitions or cluster roles: ", err)
		return errors.New("please login as cluster-admin and execute \"kamel install --cluster-setup\" to install cluster-wide resources (one-time operation)")
	} else if err != nil {
		return err
	}

	if o.clusterSetupOnly {
		if collection == nil {
			fmt.Println("Camel K cluster setup completed successfully")
		}
	} else {
		namespace := o.Namespace

		err = install.OperatorOrCollect(namespace, collection)
		if err != nil {
			return err
		}

		err = install.PlatformOrCollect(namespace, o.registry, o.organization, o.pushSecret, collection)
		if err != nil {
			return err
		}

		if o.exampleSetup {
			err = install.ExampleOrCollect(namespace, collection)
			if err != nil {
				return err
			}
		}

		if collection == nil {
			fmt.Println("Camel K installed in namespace", namespace)
		}
	}

	if collection != nil {
		return o.printOutput(collection)
	}

	return nil
}

func (o *installCmdOptions) printOutput(collection *kubernetes.Collection) error {
	lst := collection.AsKubernetesList()
	switch o.outputFormat {
	case "yaml":
		data, err := kubernetes.SerializeToYAML(lst)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	case "json":
		data, err := kubernetes.SerializeToJSON(lst)
		if err != nil {
			return err
		}
		fmt.Print(string(data))
	default:
		return errors.New("unknown output format: " + o.outputFormat)
	}
	return nil
}
