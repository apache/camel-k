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
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func TestCollectConfigurationValues(t *testing.T) {
	e := Environment{
		Integration: &v1alpha1.Integration{
			Spec: v1alpha1.IntegrationSpec{
				Configuration: []v1alpha1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-integration"},
					{Type: "env", Value: "my-env-integration"},
				},
			},
		},
		IntegrationContext: &v1alpha1.IntegrationContext{
			Spec: v1alpha1.IntegrationContextSpec{
				Configuration: []v1alpha1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-context"},
					{Type: "property", Value: "my-p-context"},
				},
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Configuration: []v1alpha1.ConfigurationSpec{
					{Type: "configmap", Value: "my-cm-platform"},
					{Type: "secret", Value: "my-secret-platform"},
					{Type: "property", Value: "my-p-platform"},
					{Type: "env", Value: "my-env-platform"},
				},
			},
		},
	}

	assert.Contains(t, e.CollectConfigurationValues("configmap"), "my-cm-integration")
	assert.Contains(t, e.CollectConfigurationValues("secret"), "my-secret-platform")
	assert.Contains(t, e.CollectConfigurationValues("property"), "my-p-context")
	assert.Contains(t, e.CollectConfigurationValues("env"), "my-env-integration")
}

func TestCollectConfigurationPairs(t *testing.T) {
	e := Environment{
		Integration: &v1alpha1.Integration{
			Spec: v1alpha1.IntegrationSpec{
				Configuration: []v1alpha1.ConfigurationSpec{
					{Type: "property", Value: "p1=integration"},
					{Type: "property", Value: "p4=integration"},
				},
			},
		},
		IntegrationContext: &v1alpha1.IntegrationContext{
			Spec: v1alpha1.IntegrationContextSpec{
				Configuration: []v1alpha1.ConfigurationSpec{
					{Type: "property", Value: "p1=context"},
					{Type: "property", Value: "p2=context"},
				},
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Configuration: []v1alpha1.ConfigurationSpec{
					{Type: "property", Value: "p1=platform"},
					{Type: "property", Value: "p2=platform"},
					{Type: "property", Value: "p3=platform"},
					{Type: "property", Value: "p4=platform"},
				},
			},
		},
	}

	pairs := e.CollectConfigurationPairs("property")
	assert.Equal(t, "integration", pairs["p1"])
	assert.Equal(t, "context", pairs["p2"])
	assert.Equal(t, "platform", pairs["p3"])
	assert.Equal(t, "integration", pairs["p4"])
}
