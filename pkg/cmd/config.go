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
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v2"
)

// ConfigFolder defines the different types of folder containing the configuration file.
type ConfigFolder string

const (
	// The path of the folder containing the configuration file is retrieved from the environment
	// variable KAMEL_CONFIG_PATH.
	ConfigFolderEnvVar ConfigFolder = "env"
	// The path of the folder containing the configuration file is $HOME/.kamel.
	ConfigFolderHome ConfigFolder = "home"
	// The folder containing the configuration file is .kamel located in the working directory.
	ConfigFolderSubDirectory ConfigFolder = "sub"
	// The folder containing the configuration file is the working directory.
	ConfigFolderWorking ConfigFolder = "working"
	// The folder containing the configuration file is the directory currently used by Kamel.
	ConfigFolderUsed ConfigFolder = "used"
)

// nolint: unparam
func newCmdConfig(rootCmdOptions *RootCmdOptions) (*cobra.Command, *configCmdOptions) {
	options := configCmdOptions{}
	cmd := cobra.Command{
		Use:     "config",
		Short:   "Configure the default settings",
		PreRunE: decode(&options),
		Args:    options.validateArgs,
		RunE:    options.run,
	}

	cmd.Flags().String("folder", "used", "The type of folder containing the configuration file to read/write. The supported values are 'env', 'home', 'sub', 'working' and 'used' for respectively $KAMEL_CONFIG_PATH, $HOME/.kamel, .kamel, . and the folder used by kamel")
	cmd.Flags().String("default-namespace", "", "The name of the namespace to use by default")
	cmd.Flags().BoolP("list", "l", false, "List all existing settings")
	return &cmd, &options
}

type configCmdOptions struct {
	DefaultNamespace string `mapstructure:"default-namespace"`
}

func (o *configCmdOptions) validateArgs(cmd *cobra.Command, args []string) error {
	if len(args) > 0 {
		return errors.New("no arguments are expected")
	}
	return nil
}

func (o *configCmdOptions) run(cmd *cobra.Command, args []string) error {
	path, err := getConfigLocation(cmd)
	if err != nil {
		return err
	}
	if cmd.Flags().Lookup("default-namespace").Changed {
		err = o.saveConfiguration(cmd, path)
		if err != nil {
			return err
		}
	}
	if cmd.Flags().Lookup("list").Changed {
		err = printConfiguration(cmd, path)
		if err != nil {
			return err
		}
	}
	return nil
}

// Save the configuration at the given location.
func (o *configCmdOptions) saveConfiguration(cmd *cobra.Command, path string) error {
	cfg, err := LoadConfigurationFrom(path)
	if err != nil {
		return err
	}

	cfg.Update(cmd, pathToRoot(cmd), o, true)

	err = cfg.Save()
	if err != nil {
		return err
	}
	return nil
}

// Gives the location of the configuration file.
func getConfigLocation(cmd *cobra.Command) (string, error) {
	var folder ConfigFolder
	if s, err := cmd.Flags().GetString("folder"); err == nil {
		folder = ConfigFolder(s)
	} else {
		return "", err
	}
	var path string
	switch folder {
	case ConfigFolderUsed:
		path = viper.ConfigFileUsed()
		if path != "" {
			return path, nil
		}
	case ConfigFolderEnvVar:
		path = os.Getenv("KAMEL_CONFIG_PATH")
	case ConfigFolderHome:
		home, err := os.UserHomeDir()
		cobra.CheckErr(err)
		path = filepath.Join(home, ".kamel")
	case ConfigFolderSubDirectory:
		path = ".kamel"
	case ConfigFolderWorking:
		path = "."
	default:
		return "", fmt.Errorf("unsupported type of folder: %s", folder)
	}
	configName := os.Getenv("KAMEL_CONFIG_NAME")
	if configName == "" {
		configName = DefaultConfigName
	}
	return filepath.Join(path, fmt.Sprintf("%s.yaml", configName)), nil
}

// Print the content of the configuration file located at the given path.
func printConfiguration(cmd *cobra.Command, path string) error {
	cfg, err := LoadConfigurationFrom(path)
	if err != nil {
		return err
	}
	if len(cfg.content) == 0 {
		fmt.Fprintf(cmd.OutOrStdout(), "No settings could be found in %s\n", cfg.location)
	} else {
		bs, err := yaml.Marshal(cfg.content)
		if err != nil {
			return fmt.Errorf("unable to marshal config to YAML: %w", err)
		}
		fmt.Fprintf(cmd.OutOrStdout(), "The configuration file is read from %s\n", cfg.location)
		fmt.Fprintln(cmd.OutOrStdout(), string(bs))
	}
	return nil
}
