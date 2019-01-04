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
	"github.com/apache/camel-k/pkg/apis"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/client/config"
	"sigs.k8s.io/controller-runtime/pkg/manager"

	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const kamelCommandLongDescription = `
Long:  "Apache Camel K (a.k.a. Kamel) is a lightweight integration framework
built from Apache Camel that runs natively on Kubernetes and is
specifically designed for serverless and microservice architectures.",,	
`

// RootCmdOptions --
type RootCmdOptions struct {
	Context    context.Context
	_client    client.Client
	KubeConfig string
	Namespace  string
}

// NewKamelCommand --
func NewKamelCommand(ctx context.Context) (*cobra.Command, error) {
	options := RootCmdOptions{
		Context: ctx,
	}
	var cmd = cobra.Command{
		Use:   "kamel",
		Short: "Kamel is a awesome client tool for running Apache Camel integrations natively on Kubernetes",
		Long:  kamelCommandLongDescription,
		BashCompletionFunction: bashCompletionFunction,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if options.Namespace == "" {
				current, err := kubernetes.GetClientCurrentNamespace(options.KubeConfig)
				if err != nil {
					return errors.Wrap(err, "cannot get current namespace")
				}
				err = cmd.Flag("namespace").Value.Set(current)
				if err != nil {
					return err
				}
			}

			// Initialize the Kubernetes client to allow using the operator-sdk
			return kubernetes.InitKubeClient(options.KubeConfig)
		},
	}

	cmd.PersistentFlags().StringVar(&options.KubeConfig, "config", "", "Path to the config file to use for CLI requests")
	cmd.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace to use for all operations")

	cmd.AddCommand(newCmdCompletion(&cmd))
	cmd.AddCommand(newCmdVersion())
	cmd.AddCommand(newCmdRun(&options))
	cmd.AddCommand(newCmdGet(&options))
	cmd.AddCommand(newCmdDelete(&options))
	cmd.AddCommand(newCmdInstall(&options))
	cmd.AddCommand(newCmdLog(&options))
	cmd.AddCommand(newCmdContext(&options))
	cmd.AddCommand(newCmdReset(&options))

	return &cmd, nil
}

// GetCmdClient returns a client that can be used from command line tools
func (command *RootCmdOptions) GetCmdClient() (client.Client, error) {
	// Get the pre-computed client
	if command._client != nil {
		return command._client, nil
	}
	// Get a config to talk to the apiserver
	cfg, err := config.GetConfig()
	if err != nil {
		return nil, err
	}
	// Create a new Cmd to provide shared dependencies and start components
	mgr, err := manager.New(cfg, manager.Options{Namespace: command.Namespace})

	// Setup Scheme for all resources
	if err := apis.AddToScheme(mgr.GetScheme()); err != nil {
		return nil, err
	}
	command._client = mgr.GetClient()
	return command._client, nil
}
