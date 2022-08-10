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
	"fmt"
	"os"
	"strings"

	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"golang.org/x/term"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	"github.com/apache/camel-k/pkg/client"
	v1 "github.com/apache/camel-k/pkg/client/camel/clientset/versioned/typed/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

const kamelCommandLongDescription = `Apache Camel K is a lightweight integration platform, born on Kubernetes, with serverless
superpowers.
`

// RootCmdOptions --.
//nolint: containedctx
type RootCmdOptions struct {
	RootContext   context.Context    `mapstructure:"-"`
	Context       context.Context    `mapstructure:"-"`
	ContextCancel context.CancelFunc `mapstructure:"-"`
	_client       client.Client      `mapstructure:"-"`
	KubeConfig    string             `mapstructure:"kube-config"`
	Namespace     string             `mapstructure:"namespace"`
	Verbose       bool               `mapstructure:"verbose" yaml:",omitempty"`
}

// NewKamelCommand --.
func NewKamelCommand(ctx context.Context) (*cobra.Command, error) {
	childCtx, childCancel := context.WithCancel(ctx)
	options := RootCmdOptions{
		RootContext:   ctx,
		Context:       childCtx,
		ContextCancel: childCancel,
	}

	cmd := kamelPreAddCommandInit(&options)
	addKamelSubcommands(cmd, &options)

	if err := addHelpSubCommands(cmd, &options); err != nil {
		return cmd, err
	}

	err := kamelPostAddCommandInit(cmd)

	return cmd, err
}

func kamelPreAddCommandInit(options *RootCmdOptions) *cobra.Command {
	cmd := cobra.Command{
		BashCompletionFunction: bashCompletionFunction,
		PersistentPreRunE:      options.preRun,
		Use:                    "kamel",
		Short:                  "Kamel is a awesome client tool for running Apache Camel integrations natively on Kubernetes",
		Long:                   kamelCommandLongDescription,
		SilenceUsage:           true,
	}

	cmd.PersistentFlags().StringVar(&options.KubeConfig, "kube-config", os.Getenv("KUBECONFIG"), "Path to the kube config file to use for CLI requests")
	cmd.PersistentFlags().StringVarP(&options.Namespace, "namespace", "n", "", "Namespace to use for all operations")
	cmd.PersistentFlags().BoolVarP(&options.Verbose, "verbose", "V", false, "Verbose logging")

	cobra.AddTemplateFunc("wrappedFlagUsages", wrappedFlagUsages)
	cmd.SetUsageTemplate(usageTemplate)

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
		if !errors.As(err, &viper.ConfigFileNotFoundError{}) {
			return err
		}
	}

	return nil
}

func addKamelSubcommands(cmd *cobra.Command, options *RootCmdOptions) {
	cmd.AddCommand(newCmdCompletion(cmd))
	cmd.AddCommand(cmdOnly(newCmdVersion(options)))
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
	cmd.AddCommand(cmdOnly(newCmdDump(options)))
	cmd.AddCommand(cmdOnly(newCmdLocal(options)))
	cmd.AddCommand(cmdOnly(newCmdBind(options)))
	cmd.AddCommand(cmdOnly(newCmdPromote(options)))
	cmd.AddCommand(newCmdKamelet(options))
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
	if !isOfflineCommand(cmd) {
		c, err := command.GetCmdClient()
		if err != nil {
			return errors.Wrap(err, "cannot get command client")
		}
		if command.Namespace == "" {
			current, err := c.GetCurrentNamespace(command.KubeConfig)
			if err != nil {
				return errors.Wrap(err, "cannot get current namespace")
			}
			err = cmd.Flag("namespace").Value.Set(current)
			if err != nil {
				return err
			}
		}
		// Check that the Kamel CLI matches that of the operator.
		// The check relies on the version reported in the IntegrationPlatform status,
		// which requires the operator is running and the IntegrationPlatform resource
		// reconciled. Hence the compatibility check is skipped for the install and the operator command.
		// Furthermore, there can be any incompatibilities, as the install command deploys
		// the operator version it's compatible with.
		if cmd.Use != builderCommand && cmd.Use != installCommand && cmd.Use != operatorCommand {
			checkAndShowCompatibilityWarning(command.Context, cmd, c, command.Namespace)
		}
	}

	return nil
}

func checkAndShowCompatibilityWarning(ctx context.Context, cmd *cobra.Command, c client.Client, namespace string) {
	operatorVersion, err := operatorVersion(ctx, c, namespace)
	if err != nil {
		if k8serrors.IsNotFound(err) {
			fmt.Fprintf(cmd.ErrOrStderr(), "No IntegrationPlatform resource in %s namespace\n", namespace)
		} else {
			fmt.Fprintf(cmd.ErrOrStderr(), "Unable to retrieve the operator version: %s\n", err.Error())
		}
	} else {
		if operatorVersion != "" && !compatibleVersions(operatorVersion, defaults.Version, cmd) {
			fmt.Fprintf(cmd.ErrOrStderr(), "You're using Camel K %s client with a %s cluster operator, it's recommended to use the same version to improve compatibility.\n\n", defaults.Version, operatorVersion)
		}
	}
}

// GetCmdClient returns the client that can be used from command line tools.
func (command *RootCmdOptions) GetCmdClient() (client.Client, error) {
	// Get the pre-computed client
	if command._client != nil {
		return command._client, nil
	}
	var err error
	command._client, err = command.NewCmdClient()
	return command._client, err
}

// GetCamelCmdClient returns a client to access the Camel resources.
func (command *RootCmdOptions) GetCamelCmdClient() (*v1.CamelV1Client, error) {
	c, err := command.GetCmdClient()
	if err != nil {
		return nil, err
	}
	return v1.NewForConfig(c.GetConfig())
}

// NewCmdClient returns a new client that can be used from command line tools.
func (command *RootCmdOptions) NewCmdClient() (client.Client, error) {
	return client.NewOutOfClusterClient(command.KubeConfig)
}

func (command *RootCmdOptions) PrintVerboseOut(cmd *cobra.Command, a ...interface{}) {
	if command.Verbose {
		fmt.Fprintln(cmd.OutOrStdout(), a...)
	}
}

func (command *RootCmdOptions) PrintfVerboseOutf(cmd *cobra.Command, format string, a ...interface{}) {
	if command.Verbose {
		fmt.Fprintf(cmd.OutOrStdout(), format, a...)
	}
}
func (command *RootCmdOptions) PrintfVerboseErrf(cmd *cobra.Command, format string, a ...interface{}) {
	if command.Verbose {
		fmt.Fprintf(cmd.ErrOrStderr(), format, a...)
	}
}

func wrappedFlagUsages(cmd *cobra.Command) string {
	width := 80
	if w, _, err := term.GetSize(0); err == nil {
		width = w
	}
	return cmd.Flags().FlagUsagesWrapped(width - 1)
}

var usageTemplate = `Usage:{{if .Runnable}}
  {{.UseLine}}{{end}}{{if .HasAvailableSubCommands}}
  {{.CommandPath}} [command]{{end}}{{if .HasAvailableSubCommands}}

Available Commands:{{range .Commands}}{{if (or .IsAvailableCommand (eq .Name "help"))}}
  {{rpad .Name .NamePadding }} {{.Short}}{{end}}{{end}}{{end}}{{if .HasAvailableLocalFlags}}

Flags:
{{ wrappedFlagUsages . | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableInheritedFlags}}

Global Flags:
{{.InheritedFlags.FlagUsages | trimTrailingWhitespaces}}{{end}}{{if .HasAvailableSubCommands}}

Use "{{.CommandPath}} [command] --help" for more information about a command.{{end}}
`
