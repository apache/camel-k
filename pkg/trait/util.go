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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// GetIntegrationContext retrieves the context set on the integration
func GetIntegrationContext(integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Spec.Context == "" {
		return nil, nil
	}

	name := integration.Spec.Context
	ctx := v1alpha1.NewIntegrationContext(integration.Namespace, name)
	err := sdk.Get(&ctx)
	return &ctx, err
}

// VisitConfigurations --
func VisitConfigurations(
	configurationType string,
	context *v1alpha1.IntegrationContext,
	integration *v1alpha1.Integration,
	consumer func(string)) {

	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				consumer(c.Value)
			}
		}
	}

	if integration != nil {
		for _, c := range integration.Spec.Configuration {
			if c.Type == configurationType {
				consumer(c.Value)
			}
		}
	}
}

// VisitKeyValConfigurations --
func VisitKeyValConfigurations(
	configurationType string,
	context *v1alpha1.IntegrationContext,
	integration *v1alpha1.Integration,
	consumer func(string, string)) {

	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					consumer(pair[0], pair[1])
				}
			}
		}
	}

	if integration != nil {
		for _, c := range integration.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					consumer(pair[0], pair[1])
				}
			}
		}
	}
}
