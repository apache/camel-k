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

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	"github.com/apache/camel-k/pkg/client"
	camelv1 "github.com/apache/camel-k/pkg/client/camel/clientset/versioned/typed/camel/v1"
)

const kamelCommandLongDescription = `Apache Camel K is a lightweight integration platform, born on Kubernetes, with serverless
superpowers.
`

// RootCmdOptions --
type RootCmdOptions struct {
	RootContext   context.Context    `mapstructure:"-"`
	Context       context.Context    `mapstructure:"-"`
	ContextCancel context.CancelFunc `mapstructure:"-"`
	_client       client.Client      `mapstructure:"-"`
	KubeConfig    string             `mapstructure:"kube-config"`
	Namespace     string             `mapstructure:"namespace"`
}

// NewKamelCommand --
func NewKamelCommand(ctx context.Context) (*cobra.Command, error) {
	childCtx, childCancel := context.WithCancel(ctx)
	options := RootCmdOptions{
		RootContext:   ctx,
		Context:       childCtx,
		ContextCancel: childCancel,
	}

	var err error
	cmd := kamelPreAddCommandInit(&options)
	addKamelSubcommands(cmd, &options)

	if err := addHelpSubCommands(cmd, &options); err != nil {
		return cmd, err
	}

	if err := addLocalSubCommands(cmd, &options); err != nil {
		return cmd, err
	}

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
	if configName == "" {
		configName = DefaultConfigName
	}

	viper.SetConfigName(configName)

	configPath := os.Getenv("KAMEL_CONFIG_PATH")
	if configPath != "" {
		// if a specific config path is set, don't add
		// default locations
		viper.AddConfigPath(configPath)
	} else {
		viper.AddConfigPath(".")
		viper.AddConfigPath(".kamel")
		viper.AddConfigPath("$HOME/.kamel")
	}

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
	cmd.AddCommand(cmdOnly(newCmdUninstall(options)))
	cmd.AddCommand(cmdOnly(newCmdLog(options)))
	cmd.AddCommand(newCmdKit(options))
	cmd.AddCommand(cmdOnly(newCmdReset(options)))
	cmd.AddCommand(newCmdDescribe(options))
	cmd.AddCommand(cmdOnly(newCmdRebuild(options)))
	cmd.AddCommand(cmdOnly(newCmdOperator()))
	cmd.AddCommand(cmdOnly(newCmdBuilder(options)))
	cmd.AddCommand(cmdOnly(newCmdInit(options)))
	cmd.AddCommand(cmdOnly(newCmdDebug(options)))
	cmd.AddCommand(cmdOnly(newCmdInspect(options)))
	cmd.AddCommand(newCmdLocal(options))
}

func addHelpSubCommands(cmd *cobra.Command, options *RootCmdOptions) error {
	cmd.InitDefaultHelpCmd()

	var helpCmd *cobra.Command
	for _, c := range cmd.Commands() {
		if c.Name() == "help" {
			helpCmd = c
			break
		}
	}

	if helpCmd == nil {
		return errors.New("could not find any configured help command")
	}
	helpCmd.Annotations = map[string]string{offlineCommandLabel: "true"}

	helpCmd.AddCommand(cmdOnly(newTraitHelpCmd(options)))

	return nil
}

func (command *RootCmdOptions) preRun(cmd *cobra.Command, _ []string) error {
	if command.Namespace == "" && !isOfflineCommand(cmd) {
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

// GetCamelCmdClient returns a client to access the Camel resources
func (command *RootCmdOptions) GetCamelCmdClient() (*camelv1.CamelV1Client, error) {
	c, err := command.GetCmdClient()
	if err != nil {
		return nil, err
	}
	return camelv1.NewForConfig(c.GetConfig())
}

// NewCmdClient returns a new client that can be used from command line tools
func (command *RootCmdOptions) NewCmdClient() (client.Client, error) {
	return client.NewOutOfClusterClient(command.KubeConfig)
}
