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
	"io/ioutil"
	"os"
	"path/filepath"
	"reflect"
	"strings"

	"github.com/apache/camel-k/pkg/util"
	p "github.com/gertd/go-pluralize"

	"github.com/spf13/cobra"
	"github.com/spf13/viper"

	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultConfigName is the default config name
	DefaultConfigName = "kamel-config"

	// DefaultConfigLocation is the main place where the kamel content is stored
	DefaultConfigLocation = DefaultConfigName + ".yaml"

	// KamelTagName ---
	KamelTagName = "kamel"

	// MapstructureTagName ---
	MapstructureTagName = "mapstructure"
)

// Config is a helper class to manipulate kamel configuration files
type Config struct {
	location string
	content  map[string]interface{}
}

// LoadConfiguration loads a kamel configuration file
func LoadConfiguration() (*Config, error) {
	config := Config{
		location: viper.ConfigFileUsed(),
		content:  make(map[string]interface{}),
	}

	if config.location == "" {
		config.location = DefaultConfigLocation
	}

	if _, err := os.Stat(config.location); os.IsNotExist(err) {
		return &config, nil
	}

	data, err := ioutil.ReadFile(config.location)
	if err != nil {
		return &config, err
	}

	err = yaml.Unmarshal(data, &config.content)
	if err != nil {
		return nil, err
	}

	return &config, nil
}

// Update ---
func (cfg *Config) Update(cmd *cobra.Command, nodeID string, data interface{}, changedOnly bool) {
	values := make(map[string]interface{})

	pl := p.NewClient()
	val := reflect.ValueOf(data).Elem()

	for i := 0; i < val.NumField(); i++ {
		field := val.Type().Field(i)
		if !field.Anonymous {
			if ktag, ok := field.Tag.Lookup(KamelTagName); ok {
				ktags := strings.Split(ktag, ",")
				if util.StringSliceExists(ktags, "omitsave") {
					continue
				}
			}

			tag, ok := field.Tag.Lookup(MapstructureTagName)
			if !ok {
				continue
			}

			flagName := pl.Singular(tag)
			if flag := cmd.Flag(flagName); flag != nil && (flag.Changed || !changedOnly) {
				if !val.Field(i).IsZero() {
					values[tag] = val.Field(i).Interface()
				}
			}
		}
	}

	if len(values) > 0 {
		cfg.SetNode(nodeID, values)
	}
}

// SetNode allows to replace a subtree with a given content
func (cfg *Config) SetNode(nodeID string, nodeValues map[string]interface{}) {
	cfg.Delete(nodeID)
	node := cfg.navigate(cfg.content, nodeID, true)

	for k, v := range nodeValues {
		node[k] = v
	}
}

// Delete allows to remove a sub tree from the kamel content
func (cfg *Config) Delete(path string) {
	leaf := cfg.navigate(cfg.content, path, false)
	for k := range leaf {
		delete(leaf, k)
	}
}

// Save ---
func (cfg *Config) Save() error {
	root := filepath.Dir(cfg.location)
	if _, err := os.Stat(root); os.IsNotExist(err) {
		if e := os.MkdirAll(root, 0o700); e != nil {
			return e
		}
	}

	data, err := yaml.Marshal(cfg.content)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(cfg.location, data, 0o644)
}

func (cfg *Config) navigate(values map[string]interface{}, prefix string, create bool) map[string]interface{} {
	nodes := strings.Split(prefix, ".")

	for _, node := range nodes {
		v := values[node]

		if v == nil {
			if create {
				v = make(map[string]interface{})
				values[node] = v
			} else {
				return nil
			}
		}

		if m, ok := v.(map[string]interface{}); ok {
			values = m
		} else if mg, ok := v.(map[interface{}]interface{}); ok {
			converted := cfg.convert(mg)
			values[node] = converted
			values = converted
		} else {
			if create {
				child := make(map[string]interface{})
				values[node] = child
				return child
			}
			return nil
		}
	}
	return values
}

func (cfg *Config) convert(m map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		if ks, ok := k.(string); ok {
			res[ks] = v
		}
	}
	return res
}
