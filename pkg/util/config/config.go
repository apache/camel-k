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

package config

import (
	"io/ioutil"
	"os"
	"strings"

	p "github.com/gertd/go-pluralize"
	yaml "gopkg.in/yaml.v2"
)

const (
	// DefaultConfigLocation is the main place where the kamel config is stored
	DefaultConfigLocation = "./kamel-config.yaml"
)

// KamelConfig is a helper class to manipulate kamel configuration files
type KamelConfig struct {
	config map[string]interface{}
}

// LoadDefault loads the kamel configuration from the default location
func LoadDefault() (*KamelConfig, error) {
	return LoadConfig(DefaultConfigLocation)
}

// LoadConfig loads a kamel configuration file
func LoadConfig(file string) (*KamelConfig, error) {
	config := make(map[string]interface{})
	data, err := ioutil.ReadFile(file)
	if err != nil && os.IsNotExist(err) {
		return &KamelConfig{config: config}, nil
	} else if err != nil {
		return nil, err
	}
	if err = yaml.Unmarshal(data, &config); err != nil {
		return nil, err
	}

	return &KamelConfig{config: config}, nil
}

// Set allows to replace a subtree with a given config
func (c *KamelConfig) Set(values map[string]interface{}, from, to string, filter func(string) bool) {
	source := navigate(values, from, false)
	destination := navigate(c.config, to, true)
	pl := p.NewClient()
	for k, v := range source {
		if filter(k) {
			plural := pl.Plural(k)
			key := k
			if source[plural] != nil {
				// prefer plural names if available
				key = plural
			}
			destination[key] = v
		}
	}
}

// Delete allows to remove a substree from the kamel config
func (c *KamelConfig) Delete(path string) {
	leaf := navigate(c.config, path, false)
	for k := range leaf {
		delete(leaf, k)
	}
}

// WriteDefault writes the configuration in the default location
func (c *KamelConfig) WriteDefault() error {
	return c.Write(DefaultConfigLocation)
}

// Write writes a kamel configuration to a file
func (c *KamelConfig) Write(file string) error {
	data, err := yaml.Marshal(c.config)
	if err != nil {
		return err
	}
	return ioutil.WriteFile(file, data, 0777)
}

func navigate(values map[string]interface{}, prefix string, create bool) map[string]interface{} {
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
			converted := convert(mg)
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

func convert(m map[interface{}]interface{}) map[string]interface{} {
	res := make(map[string]interface{})
	for k, v := range m {
		if ks, ok := k.(string); ok {
			res[ks] = v
		}
	}
	return res
}
