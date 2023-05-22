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
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/maven"
	"github.com/apache/camel-k/v2/pkg/util/resource"
	"github.com/magiconair/properties"
	"github.com/pkg/errors"
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
			return fmt.Errorf("you cannot provide a binary config, use a text file instead")
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

func loadPropertiesFromSecret(ctx context.Context, c client.Client, ns string, name string) (*properties.Properties, error) {
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

func loadPropertiesFromConfigMap(ctx context.Context, c client.Client, ns string, name string) (*properties.Properties, error) {
	cm := kubernetes.LookupConfigmap(ctx, c, ns, name)
	if cm == nil {
		return nil, fmt.Errorf("%s configmap not found in %s namespace, make sure to provide it before the Integration can run", name, ns)
	}
	return fromMapToProperties(cm.Data,
		func(v reflect.Value) string { return v.String() },
		func(v reflect.Value) (*properties.Properties, error) { return keyValueProps(v.String()) })
}

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

// downloadDependency downloads the file located at the given URL into a temporary folder and returns the local path to the generated temporary file.
func downloadDependency(ctx context.Context, url url.URL) (string, error) {
	tctx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(tctx, http.MethodGet, url.String(), nil)
	if err != nil {
		return "", err
	}
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	base := filepath.Base(url.Path)
	if base == "." || base == "/" || filepath.Ext(base) == "" {
		base = filepath.Base(url.String())
		if base == "." || base == "/" {
			base = "tmp"
		}
	}
	out, err := os.CreateTemp("", fmt.Sprintf("*.%s", base))
	if err != nil {
		return "", err
	}
	defer out.Close()
	_, err = io.Copy(out, res.Body)
	if err != nil {
		return "", err
	}
	return out.Name(), nil
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
		return errors.Wrapf(err, "unable to access property file %s", fileName)
	} else if file.IsDir() {
		return fmt.Errorf("property file %s is a directory", fileName)
	}

	return nil
}

func createCamelCatalog(ctx context.Context) (*camel.RuntimeCatalog, error) {
	// Attempt to reuse existing Camel catalog if one is present
	catalog, err := camel.DefaultCatalog()
	if err != nil {
		return nil, err
	}

	// Generate catalog if one was not found
	if catalog == nil {
		catalog, err = generateCatalog(ctx)
		if err != nil {
			return nil, err
		}
	}

	return catalog, nil
}

func generateCatalog(ctx context.Context) (*camel.RuntimeCatalog, error) {
	// A Camel catalog is required for this operation
	mvn := v1.MavenSpec{
		LocalRepository: "",
	}
	runtime := v1.RuntimeSpec{
		Version:  defaults.DefaultRuntimeVersion,
		Provider: v1.RuntimeProviderQuarkus,
	}
	var providerDependencies []maven.Dependency
	var caCert [][]byte
	catalog, err := camel.GenerateCatalogCommon(ctx, nil, nil, caCert, mvn, runtime, providerDependencies)
	if err != nil {
		return nil, err
	}

	return catalog, nil
}
