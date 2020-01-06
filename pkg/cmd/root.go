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
	"os"
	"strings"

	"github.com/spf13/viper"

	"github.com/apache/camel-k/pkg/client"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
)

const kamelCommandLongDescription = `Apache Camel K is a lightweight integration platform, born on Kubernetes, with serverless
superpowers.
`

// RootCmdOptions --
type RootCmdOptions struct {
	Context    context.Context `mapstructure:"-"`
	_client    client.Client   `mapstructure:"-"`
	KubeConfig string          `mapstructure:"kube-config"`
	Namespace  string          `mapstructure:"namespace"`
}

// NewKamelCommand --
func NewKamelCommand(ctx context.Context) (*cobra.Command, error) {
	options := RootCmdOptions{
		Context: ctx,
	}

	var err error
	cmd := kamelPreAddCommandInit(&options)
	addKamelSubcommands(cmd, &options)
	err = kamelPostAddCommandInit(cmd)

	return cmd, err
}

func kamelPreAddCommandInit(options *RootCmdOptions) *cobra.Command {

	var cmd = cobra.Command{
		BashCompletionFunction: bashCompletionFunction,
		PersistentPreRunE:      options.preRun,
		Use:                    "kamel",
		Short:                  "Kamel is a awesome client tool for running Apache Camel integrations natively on Kubernetes",
		Long:                   kamelCommandLongDescription,
		SilenceUsage:           true,
	}

	cmd.PersistentFlags().StringVar(&options.KubeConfig, "config", os.Getenv("KUBECONFIG"), "Path to the config file to use for CLI requests")
	cmd.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace to use for all operations")

	return &cmd
}

func kamelPostAddCommandInit(cmd *cobra.Command) error {
	if err := bindPFlagsHierarchy(cmd); err != nil {
		return err
	}

	configName := os.Getenv("KAMEL_CONFIG_NAME")
	if configName != "" {
		configName = "config"
	}

	viper.SetConfigName(configName)
	viper.AddConfigPath(".kamel")
	viper.AddConfigPath("$HOME/.kamel")
	viper.AutomaticEnv()
	viper.SetEnvKeyReplacer(strings.NewReplacer(
		".", "_",
		"-", "_",
	))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return err
		}
	}

	return nil
}

func addKamelSubcommands(cmd *cobra.Command, options *RootCmdOptions) {
	cmd.AddCommand(newCmdCompletion(cmd))
	cmd.AddCommand(newCmdVersion())
	cmd.AddCommand(cmdOnly(newCmdRun(options)))
	cmd.AddCommand(cmdOnly(newCmdGet(options)))
	cmd.AddCommand(cmdOnly(newCmdDelete(options)))
	cmd.AddCommand(cmdOnly(newCmdInstall(options)))
	cmd.AddCommand(cmdOnly(newCmdLog(options)))
	cmd.AddCommand(newCmdKit(options))
	cmd.AddCommand(cmdOnly(newCmdReset(options)))
	cmd.AddCommand(newCmdDescribe(options))
	cmd.AddCommand(cmdOnly(newCmdRebuild(options)))
	cmd.AddCommand(newCmdOperator())
	cmd.AddCommand(cmdOnly(newCmdBuilder(options)))
}

func (command *RootCmdOptions) preRun(cmd *cobra.Command, _ []string) error {
	if command.Namespace == "" {
		var current string
		client, err := command.GetCmdClient()
		if err != nil {
			return errors.Wrap(err, "cannot get command client")
		}
		current, err = client.GetCurrentNamespace(command.KubeConfig)
		if err != nil {
			return errors.Wrap(err, "cannot get current namespace")
		}
		err = cmd.Flag("namespace").Value.Set(current)
		if err != nil {
			return err
		}
	}
	return nil
}

// GetCmdClient returns the client that can be used from command line tools
func (command *RootCmdOptions) GetCmdClient() (client.Client, error) {
	// Get the pre-computed client
	if command._client != nil {
		return command._client, nil
	}
	var err error
	command._client, err = command.NewCmdClient()
	return command._client, err
}

// NewCmdClient returns a new client that can be used from command line tools
func (command *RootCmdOptions) NewCmdClient() (client.Client, error) {
	return client.NewOutOfClusterClient(command.KubeConfig)
}
