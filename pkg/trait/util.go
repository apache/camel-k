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
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
)

// GetIntegrationContext retrieves the context set on the integration
func GetIntegrationContext(integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Spec.Context == "" {
		return nil, errors.New("no context set on the integration")
	}

	name := integration.Spec.Context
	ctx := v1alpha1.NewIntegrationContext(integration.Namespace, name)
	err := sdk.Get(&ctx)
	return &ctx, err
}

// PropertiesString --
func PropertiesString(m map[string]string) string {
	properties := ""
	for k, v := range m {
		properties += fmt.Sprintf("%s=%s\n", k, v)
	}

	return properties
}

// EnvironmentAsEnvVarSlice --
func EnvironmentAsEnvVarSlice(m map[string]string) []v1.EnvVar {
	env := make([]v1.EnvVar, 0, len(m))

	for k, v := range m {
		env = append(env, v1.EnvVar{Name: k, Value: v})
	}

	return env
}

// CombineConfigurationAsMap --
func CombineConfigurationAsMap(configurationType string, context *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) map[string]string {
	result := make(map[string]string)
	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					result[pair[0]] = pair[1]
				}
			}
		}
	}

	if integration != nil {
		for _, c := range integration.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					result[pair[0]] = pair[1]
				}
			}
		}
	}

	return result
}

// CombineConfigurationAsSlice --
func CombineConfigurationAsSlice(configurationType string, context *v1alpha1.IntegrationContext, integration *v1alpha1.Integration) []string {
	result := make(map[string]bool, 0)
	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				result[c.Value] = true
			}
		}
	}

	for _, c := range integration.Spec.Configuration {
		if c.Type == configurationType {
			result[c.Value] = true
		}
	}

	keys := make([]string, 0, len(result))
	for k := range result {
		keys = append(keys, k)
	}

	return keys
}

