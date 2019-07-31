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

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetIntegrationContext retrieves the context set on the integration
func GetIntegrationContext(ctx context.Context, c client.Client, integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Status.Context == "" {
		return nil, nil
	}

	name := integration.Status.Context
	ictx := v1alpha1.NewIntegrationContext(integration.Namespace, name)
	key := k8sclient.ObjectKey{
		Namespace: integration.Namespace,
		Name:      name,
	}
	err := c.Get(ctx, key, &ictx)
	return &ictx, err
}

// CollectConfigurationValues --
func CollectConfigurationValues(configurationType string, configurable ...v1alpha1.Configurable) []string {
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
func CollectConfigurationPairs(configurationType string, configurable ...v1alpha1.Configurable) map[string]string {
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

var (
	csvMapValidatingRegexp = regexp.MustCompile(`^(\w+)=([^,]+)(?:,(\w+)=([^,]+))*$`)
	csvMapParsingRegexp    = regexp.MustCompile(`(\w+)=([^,]+)`)
)

func parseCsvMap(csvMap *string) (map[string]string, error) {
	m := make(map[string]string)

	if csvMap == nil || len(*csvMap) == 0 {
		return m, nil
	}

	if !csvMapValidatingRegexp.MatchString(*csvMap) {
		return nil, fmt.Errorf("cannot parse [%s] as CSV map", *csvMap)
	}

	matches := csvMapParsingRegexp.FindAllStringSubmatch(*csvMap, -1)
	for i := range matches {
		m[matches[i][1]] = matches[i][2]
	}

	return m, nil
}
