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

package platform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
)

func TestFindIntegrationProfile(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "ns",
		},
	}

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "custom",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

func TestFindIntegrationProfileWithNamespace(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "other",
		},
	}

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation:          "custom",
				v1.IntegrationProfileNamespaceAnnotation: "other",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.NotNil(t, found)
}

func TestFindIntegrationProfileInOperatorNamespace(t *testing.T) {
	profile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "custom",
			Namespace: "operator-namespace",
		},
	}

	c, err := internal.NewFakeClient(&profile)
	require.NoError(t, err)

	t.Setenv(operatorNamespaceEnvVariable, "operator-namespace")

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "custom",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.NotNil(t, found)
}
