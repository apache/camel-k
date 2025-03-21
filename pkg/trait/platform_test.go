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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/camel-k/v2/pkg/internal"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

func TestPlatformTraitChangeStatus(t *testing.T) {
	table := []struct {
		name         string
		initialPhase v1.IntegrationPhase
	}{
		{
			name:         "Setup from [none]",
			initialPhase: v1.IntegrationPhaseNone,
		},
		{
			name:         "Setup from WaitingForPlatform",
			initialPhase: v1.IntegrationPhaseWaitingForPlatform,
		},
	}

	for _, entry := range table {
		input := entry
		t.Run(input.name, func(t *testing.T) {
			e := Environment{
				Resources: kubernetes.NewCollection(),
				Integration: &v1.Integration{
					Status: v1.IntegrationStatus{
						Phase: input.initialPhase,
					},
				},
			}

			trait, _ := newPlatformTrait().(*platformTrait)
			trait.CreateDefault = ptr.To(false)

			var err error
			trait.Client, err = internal.NewFakeClient()
			require.NoError(t, err)

			enabled, condition, err := trait.Configure(&e)
			require.NoError(t, err)
			assert.True(t, enabled)
			assert.Nil(t, condition)

			err = trait.Apply(&e)
			require.NoError(t, err)

			assert.Equal(t, v1.IntegrationPhaseWaitingForPlatform, e.Integration.Status.Phase)
			assert.Empty(t, e.Resources.Items())
		})
	}
}

func TestPlatformTraitCreatesDefaultPlatform(t *testing.T) {
	e := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns1",
				Name:      "xx",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseNone,
			},
		},
	}

	trait, _ := newPlatformTrait().(*platformTrait)
	trait.CreateDefault = ptr.To(true)

	var err error
	trait.Client, err = internal.NewFakeClient()
	require.NoError(t, err)

	enabled, condition, err := trait.Configure(&e)
	require.NoError(t, err)
	assert.True(t, enabled)
	assert.Nil(t, condition)

	err = trait.Apply(&e)
	require.NoError(t, err)

	assert.Equal(t, v1.IntegrationPhaseWaitingForPlatform, e.Integration.Status.Phase)
	assert.Equal(t, 1, len(e.Resources.Items()))
	defPlatform := v1.NewIntegrationPlatform("ns1", platform.DefaultPlatformName)
	defPlatform.Labels = map[string]string{"app": "camel-k", "camel.apache.org/platform.generated": boolean.TrueString}
	assert.Contains(t, e.Resources.Items(), &defPlatform)
}

func TestPlatformTraitExisting(t *testing.T) {
	table := []struct {
		name          string
		platformPhase v1.IntegrationPlatformPhase
		expectedPhase v1.IntegrationPhase
	}{
		{
			name:          "Wait existing",
			platformPhase: "",
			expectedPhase: v1.IntegrationPhaseWaitingForPlatform,
		},
		{
			name:          "Move state",
			platformPhase: v1.IntegrationPlatformPhaseReady,
			expectedPhase: v1.IntegrationPhaseNone,
		},
	}

	for _, entry := range table {
		input := entry
		t.Run(input.name, func(t *testing.T) {
			e := Environment{
				Resources: kubernetes.NewCollection(),
				Integration: &v1.Integration{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: "ns1",
						Name:      "xx",
					},
					Status: v1.IntegrationStatus{
						Phase: v1.IntegrationPhaseNone,
					},
				},
			}

			trait, _ := newPlatformTrait().(*platformTrait)
			trait.CreateDefault = ptr.To(true)

			var err error
			existingPlatform := v1.NewIntegrationPlatform("ns1", "existing")
			existingPlatform.Status.Phase = input.platformPhase
			trait.Client, err = internal.NewFakeClient(&existingPlatform)
			require.NoError(t, err)

			enabled, condition, err := trait.Configure(&e)
			require.NoError(t, err)
			assert.True(t, enabled)
			assert.Nil(t, condition)

			err = trait.Apply(&e)
			require.NoError(t, err)

			assert.Equal(t, input.expectedPhase, e.Integration.Status.Phase)
			assert.Empty(t, e.Resources.Items())
		})
	}
}
