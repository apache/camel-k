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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestPlatformTraitChangeStatus(t *testing.T) {

	table := []struct {
		name         string
		initialPhase v1alpha1.IntegrationPhase
	}{
		{
			name:         "Setup from [none]",
			initialPhase: v1alpha1.IntegrationPhaseNone,
		},
		{
			name:         "Setup from WaitingForPlatform",
			initialPhase: v1alpha1.IntegrationPhaseWaitingForPlatform,
		},
	}

	for _, entry := range table {
		input := entry
		t.Run(input.name, func(t *testing.T) {
			e := Environment{
				Resources: kubernetes.NewCollection(),
				Integration: &v1alpha1.Integration{
					Status: v1alpha1.IntegrationStatus{
						Phase: input.initialPhase,
					},
				},
			}

			trait := newPlatformTrait()
			createPlatform := false
			trait.CreateDefault = &createPlatform

			var err error
			trait.client, err = test.NewFakeClient()
			assert.Nil(t, err)

			enabled, err := trait.Configure(&e)
			assert.Nil(t, err)
			assert.True(t, enabled)

			err = trait.Apply(&e)
			assert.Nil(t, err)

			assert.Equal(t, v1alpha1.IntegrationPhaseWaitingForPlatform, e.Integration.Status.Phase)
			assert.Empty(t, e.Resources.Items())
		})
	}
}

func TestPlatformTraitCreatesDefaultPlatform(t *testing.T) {
	e := Environment{
		Resources: kubernetes.NewCollection(),
		Integration: &v1alpha1.Integration{
			ObjectMeta: v1.ObjectMeta{
				Namespace: "ns1",
				Name:      "xx",
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseNone,
			},
		},
	}

	trait := newPlatformTrait()
	createPlatform := true
	trait.CreateDefault = &createPlatform

	var err error
	trait.client, err = test.NewFakeClient()
	assert.Nil(t, err)

	enabled, err := trait.Configure(&e)
	assert.Nil(t, err)
	assert.True(t, enabled)

	err = trait.Apply(&e)
	assert.Nil(t, err)

	assert.Equal(t, v1alpha1.IntegrationPhaseWaitingForPlatform, e.Integration.Status.Phase)
	assert.Equal(t, 1, len(e.Resources.Items()))
	defPlatform := v1alpha1.NewIntegrationPlatform("ns1", platform.DefaultPlatformName)
	assert.Contains(t, e.Resources.Items(), &defPlatform)
}

func TestPlatformTraitExisting(t *testing.T) {

	table := []struct {
		name          string
		platformPhase v1alpha1.IntegrationPlatformPhase
		expectedPhase v1alpha1.IntegrationPhase
	}{
		{
			name:          "Wait existing",
			platformPhase: "",
			expectedPhase: v1alpha1.IntegrationPhaseWaitingForPlatform,
		},
		{
			name:          "Move state",
			platformPhase: v1alpha1.IntegrationPlatformPhaseReady,
			expectedPhase: v1alpha1.IntegrationPhaseInitialization,
		},
	}

	for _, entry := range table {
		input := entry
		t.Run(input.name, func(t *testing.T) {
			e := Environment{
				Resources: kubernetes.NewCollection(),
				Integration: &v1alpha1.Integration{
					ObjectMeta: v1.ObjectMeta{
						Namespace: "ns1",
						Name:      "xx",
					},
					Status: v1alpha1.IntegrationStatus{
						Phase: v1alpha1.IntegrationPhaseNone,
					},
				},
			}

			trait := newPlatformTrait()
			createPlatform := true
			trait.CreateDefault = &createPlatform

			var err error
			existingPlatform := v1alpha1.NewIntegrationPlatform("ns1", "existing")
			existingPlatform.Status.Phase = input.platformPhase
			trait.client, err = test.NewFakeClient(&existingPlatform)
			assert.Nil(t, err)

			enabled, err := trait.Configure(&e)
			assert.Nil(t, err)
			assert.True(t, enabled)

			err = trait.Apply(&e)
			assert.Nil(t, err)

			assert.Equal(t, input.expectedPhase, e.Integration.Status.Phase)
			assert.Empty(t, e.Resources.Items())
		})
	}
}
