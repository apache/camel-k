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
	"strings"

	user "github.com/mitchellh/go-homedir"
	"github.com/scylladb/go-set/strset"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
)

var exactVersionRegexp = regexp.MustCompile(`^(\d+)\.(\d+)\.([\w-.]+)$`)

func BoolP(b bool) *bool {
	return &b
}

func IntP(i int) *int {
	return &i
}

// GetIntegrationKit retrieves the kit set on the integration
func GetIntegrationKit(ctx context.Context, c client.Client, integration *v1.Integration) (*v1.IntegrationKit, error) {
	if integration.Status.Kit == "" {
		return nil, nil
	}

	name := integration.Status.Kit
	kit := v1.NewIntegrationKit(integration.Namespace, name)
	key := k8sclient.ObjectKey{
		Namespace: integration.Namespace,
		Name:      name,
	}
	err := c.Get(ctx, key, &kit)
	return &kit, err
}

// CollectConfigurationValues --
func CollectConfigurationValues(configurationType string, configurable ...v1.Configurable) []string {
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

	return result.List()
}

// CollectConfigurationPairs --
func CollectConfigurationPairs(configurationType string, configurable ...v1.Configurable) map[string]string {
	result := make(map[string]string)

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
				pair := strings.SplitN(entry.Value, "=", 2)
				if len(pair) == 2 {
					k := strings.TrimSpace(pair[0])
					v := strings.TrimSpace(pair[1])

					if len(k) > 0 && len(v) > 0 {
						result[k] = v
					}
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

// FilterTransferableAnnotations returns a map containing annotations that are meaningful for being transferred to child resources.
func FilterTransferableAnnotations(annotations map[string]string) map[string]string {
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
