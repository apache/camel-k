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

package trait

import (
	"context"
	"fmt"
	"reflect"
	"regexp"
	"sort"
	"strings"

	user "github.com/mitchellh/go-homedir"
	"github.com/scylladb/go-set/strset"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/property"
)

var exactVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)\.([\w-.]+)$`)

// getIntegrationKit retrieves the kit set on the integration.
func getIntegrationKit(ctx context.Context, c client.Client, integration *v1.Integration) (*v1.IntegrationKit, error) {
	if integration.Status.IntegrationKit == nil {
		return nil, nil
	}
	kit := v1.NewIntegrationKit(integration.Status.IntegrationKit.Namespace, integration.Status.IntegrationKit.Name)
	err := c.Get(ctx, ctrl.ObjectKeyFromObject(kit), kit)
	return kit, err
}

func collectConfigurationValues(configurationType string, configurable ...v1.Configurable) []string {
	result := strset.New()

	for _, c := range configurable {
		c := c

		if c == nil || reflect.ValueOf(c).IsNil() {
			continue
		}

		entries := c.Configurations()
		if entries == nil {
			continue
		}

		for _, entry := range entries {
			if entry.Type == configurationType {
				result.Add(entry.Value)
			}
		}
	}

	s := result.List()
	sort.Strings(s)
	return s
}

func collectConfigurations(configurationType string, configurable ...v1.Configurable) []map[string]string {
	var result []map[string]string

	for _, c := range configurable {
		c := c

		if c == nil || reflect.ValueOf(c).IsNil() {
			continue
		}

		entries := c.Configurations()
		if entries == nil {
			continue
		}

		// nolint: staticcheck
		for _, entry := range entries {
			if entry.Type == configurationType {
				item := make(map[string]string)
				item["value"] = entry.Value
				item["resourceType"] = entry.ResourceType
				item["resourceMountPoint"] = entry.ResourceMountPoint
				item["resourceKey"] = entry.ResourceKey
				result = append(result, item)
			}
		}
	}

	return result
}

func collectConfigurationPairs(configurationType string, configurable ...v1.Configurable) []variable {
	result := make([]variable, 0)

	for _, c := range configurable {
		c := c

		if c == nil || reflect.ValueOf(c).IsNil() {
			continue
		}

		entries := c.Configurations()
		if entries == nil {
			continue
		}

		for _, entry := range entries {
			if entry.Type == configurationType {
				k, v := property.SplitPropertyFileEntry(entry.Value)
				if k == "" {
					continue
				}

				ok := false
				for i, variable := range result {
					if variable.Name == k {
						result[i].Value = v
						ok = true
						break
					}
				}
				if !ok {
					result = append(result, variable{Name: k, Value: v})
				}
			}
		}
	}

	return result
}

var keyValuePairRegexp = regexp.MustCompile(`^(\w+)=(.+)$`)

func keyValuePairArrayAsStringMap(pairs []string) (map[string]string, error) {
	m := make(map[string]string)

	for _, pair := range pairs {
		if match := keyValuePairRegexp.FindStringSubmatch(pair); match != nil {
			m[match[1]] = match[2]
		} else {
			return nil, fmt.Errorf("unable to parse key/value pair: %s", pair)
		}
	}

	return m, nil
}

// filterTransferableAnnotations returns a map containing annotations that are meaningful for being transferred to child resources.
func filterTransferableAnnotations(annotations map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range annotations {
		if strings.HasPrefix(k, "kubectl.kubernetes.io") {
			// filter out kubectl annotations
			continue
		}
		res[k] = v
	}
	return res
}

func mustHomeDir() string {
	dir, err := user.Dir()
	if err != nil {
		panic(err)
	}
	return dir
}

func toHostDir(host string) string {
	h := strings.Replace(strings.Replace(host, "https://", "", 1), "http://", "", 1)
	return toFileName.ReplaceAllString(h, "_")
}

func AddSourceDependencies(source v1.SourceSpec, catalog *camel.RuntimeCatalog) *strset.Set {
	dependencies := strset.New()

	// Add auto-detected dependencies
	meta := metadata.Extract(catalog, source)
	dependencies.Merge(meta.Dependencies)

	// Add loader dependencies
	lang := source.InferLanguage()
	for loader, v := range catalog.Loaders {
		// add loader specific dependencies
		if source.Loader != "" && source.Loader == loader {
			dependencies.Add(v.GetDependencyID())

			for _, d := range v.Dependencies {
				dependencies.Add(d.GetDependencyID())
			}
		} else if source.Loader == "" {
			// add language specific dependencies
			if util.StringSliceExists(v.Languages, string(lang)) {
				dependencies.Add(v.GetDependencyID())

				for _, d := range v.Dependencies {
					dependencies.Add(d.GetDependencyID())
				}
			}
		}
	}

	return dependencies
}
