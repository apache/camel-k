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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestIntegrationExecutedTrait(t *testing.T) {
	ip := v1.IntegrationPlatform{}
	ip.Namespace = "ns"
	ip.Name = "ck"
	ip.Status = v1.IntegrationPlatformStatus{
		IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
			Build: v1.IntegrationPlatformBuildSpec{
				RuntimeProvider: v1.RuntimeProviderQuarkus,
				RuntimeVersion:  defaults.DefaultRuntimeVersion,
			},
		},
		Phase: v1.IntegrationPlatformPhaseReady,
	}
	c, err := internal.NewFakeClient(&ip)
	require.NoError(t, err)
	catalog := NewCatalog(c)
	env := Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Runtime: v1.RuntimeSpec{
					Version:  defaults.DefaultRuntimeVersion,
					Provider: v1.RuntimeProviderQuarkus,
				},
			},
		},
		Catalog: catalog,
		Ctx:     context.Background(),
		Client:  c,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "it",
				Namespace: "ns",
			},
			Spec: v1.IntegrationSpec{
				Profile: v1.TraitProfileKubernetes,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
				Conditions: []v1.IntegrationCondition{
					{
						Type:   v1.IntegrationConditionDeploymentAvailable,
						Status: corev1.ConditionTrue,
					},
				},
			},
		},
		Platform:  &ip,
		Resources: kubernetes.NewCollection(),
	}

	_, ts, err := catalog.apply(&env)
	require.NoError(t, err)
	assert.Equal(t, &v1.Traits{}, ts)
}
