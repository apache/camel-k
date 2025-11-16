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
	"errors"
	"fmt"
	"os"
	"reflect"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/resource"
	"github.com/magiconair/properties"
	"github.com/spf13/cobra"
)

func addDependency(cmd *cobra.Command, it *v1.Integration, dependency string, catalog *camel.RuntimeCatalog) {
	normalized := camel.NormalizeDependency(dependency)
	camel.ValidateDependency(catalog, normalized, cmd.ErrOrStderr())
	it.Spec.AddDependency(normalized)
}

func parseConfig(ctx context.Context, cmd *cobra.Command, c client.Client, config *resource.Config, integration *v1.Integration) error {
	switch config.StorageType() {
	case resource.StorageTypeConfigmap:
		cm := kubernetes.LookupConfigmap(ctx, c, integration.Namespace, config.Name())
		if cm == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Warn:", config.Name(), "Configmap not found in", integration.Namespace, "namespace, make sure to provide it before the Integration can run")
		} else if config.ContentType() != resource.ContentTypeData && cm.BinaryData != nil {
			return errors.New("you cannot provide a binary config, use a text file instead")
		}
	case resource.StorageTypeSecret:
		secret := kubernetes.LookupSecret(ctx, c, integration.Namespace, config.Name())
		if secret == nil {
			fmt.Fprintln(cmd.ErrOrStderr(), "Warn:", config.Name(), "Secret not found in", integration.Namespace, "namespace, make sure to provide it before the Integration can run")
		}
	default:
		// Should never reach this
		return fmt.Errorf("invalid option type %s", config.StorageType())
	}

	return nil
}

func filterFileLocation(maybeFileLocations []string) []string {
	filteredOptions := make([]string, 0)
	for _, option := range maybeFileLocations {
		if strings.HasPrefix(option, "file:") {
			localPath, _ := resource.ParseFileValue(strings.Replace(option, "file:", "", 1))
			filteredOptions = append(filteredOptions, localPath)
		}
	}

	return filteredOptions
}

func keyValueProps(value string) (*properties.Properties, error) {
	return properties.Load([]byte(value), properties.UTF8)
}

// Deprecated: won't be supported in future releases.
func loadPropertiesFromSecret(ctx context.Context, c client.Client, ns string, name string) (*properties.Properties, error) {
	if c == nil {
		return nil, errors.New("cannot inspect Secrets in offline mode")
	}
	secret := kubernetes.LookupSecret(ctx, c, ns, name)
	if secret == nil {
		return nil, fmt.Errorf("%s secret not found in %s namespace, make sure to provide it before the Integration can run", name, ns)
	}

	return fromMapToProperties(secret.Data,
		func(v reflect.Value) string { return string(v.Bytes()) },
		func(v reflect.Value) (*properties.Properties, error) {
			return properties.Load(v.Bytes(), properties.UTF8)
		})
}

// Deprecated: won't be supported in future releases.
func loadPropertiesFromConfigMap(ctx context.Context, c client.Client, ns string, name string) (*properties.Properties, error) {
	if c == nil {
		return nil, errors.New("cannot inspect Configmaps in offline mode")
	}
	cm := kubernetes.LookupConfigmap(ctx, c, ns, name)
	if cm == nil {
		return nil, fmt.Errorf("%s configmap not found in %s namespace, make sure to provide it before the Integration can run", name, ns)
	}

	return fromMapToProperties(cm.Data,
		func(v reflect.Value) string { return v.String() },
		func(v reflect.Value) (*properties.Properties, error) { return keyValueProps(v.String()) })
}

// Deprecated: func supporting other deprecated funcs.
func fromMapToProperties(data interface{}, toString func(reflect.Value) string, loadProperties func(reflect.Value) (*properties.Properties, error)) (*properties.Properties, error) {
	result := properties.NewProperties()
	m := reflect.ValueOf(data)
	for _, k := range m.MapKeys() {
		key := k.String()
		value := m.MapIndex(k)
		if strings.HasSuffix(key, ".properties") {
			p, err := loadProperties(value)
			if err == nil {
				result.Merge(p)
			} else if _, _, err = result.Set(key, toString(value)); err != nil {
				return nil, fmt.Errorf("cannot assign %s to %s", value, key)
			}
		} else if _, _, err := result.Set(key, toString(value)); err != nil {
			return nil, fmt.Errorf("cannot assign %s to %s", value, key)
		}
	}

	return result, nil
}

func validatePropertyFiles(propertyFiles []string) error {
	for _, fileName := range propertyFiles {
		if err := validatePropertyFile(fileName); err != nil {
			return err
		}
	}

	return nil
}

func validatePropertyFile(fileName string) error {
	if !strings.HasSuffix(fileName, ".properties") {
		return fmt.Errorf("supported property files must have a .properties extension: %s", fileName)
	}

	if file, err := os.Stat(fileName); err != nil {
		return fmt.Errorf("unable to access property file %s", fileName)
	} else if file.IsDir() {
		return fmt.Errorf("property file %s is a directory", fileName)
	}

	return nil
}

func createCamelCatalog() (*camel.RuntimeCatalog, error) {
	// Attempt to reuse existing Camel catalog if one is present
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}

	return catalog, nil
}

func extractTraitNames(traitProps []string) []string {
	traitNameProps := make([]string, len(traitProps))
	for i, tp := range traitProps {
		splits := strings.Split(tp, ".")
		traitNameProps[i] = splits[0]
	}

	return traitNameProps
}
