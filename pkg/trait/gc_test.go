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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"
)

func TestConfigureGCTraitDoesSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	configured, err := gcTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigureDisabledGCTraitDoesNotSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	gcTrait.Enabled = pointer.Bool(false)

	configured, err := gcTrait.Configure(environment)

	assert.False(t, configured)
	assert.Nil(t, err)
}

func TestApplyGarbageCollectorTraitFirstGenerationDoesSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()

	err := gcTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.PostProcessors, 1)
	assert.Len(t, environment.PostActions, 0)
}

func TestApplyGarbageCollectorTraitNextGenerationDoesSucceed(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Generation = 2

	err := gcTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.PostProcessors, 1)
	assert.Len(t, environment.PostActions, 1)
}

func TestApplyGCTraitDuringInitializationPhaseSkipPostActions(t *testing.T) {
	gcTrait, environment := createNominalGCTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	err := gcTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.PostProcessors, 1)
	assert.Len(t, environment.PostActions, 0)
}

func createNominalGCTest() (*gcTrait, *Environment) {
	trait, _ := newGCTrait().(*gcTrait)
	trait.Enabled = pointer.Bool(true)

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:       "integration-name",
				Generation: 1,
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}

	return trait, environment
}
