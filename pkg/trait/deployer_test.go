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
)

func TestConfigureDeployerTraitDoesSucceed(t *testing.T) {
	deployerTrait, environment := createNominalDeployerTest()

	configured, err := deployerTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestConfigureDeployerTraitInWrongPhaseDoesNotSucceed(t *testing.T) {
	deployerTrait, environment := createNominalDeployerTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseError

	configured, err := deployerTrait.Configure(environment)

	assert.True(t, configured)
	assert.Nil(t, err)
}

func TestApplyDeployerTraitDoesSucceed(t *testing.T) {
	deployerTrait, environment := createNominalDeployerTest()

	err := deployerTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.PostActions, 1)
}

func TestApplyDeployerTraitInInitializationPhaseDoesSucceed(t *testing.T) {
	deployerTrait, environment := createNominalDeployerTest()
	environment.Integration.Status.Phase = v1.IntegrationPhaseInitialization

	err := deployerTrait.Apply(environment)

	assert.Nil(t, err)
	assert.Len(t, environment.PostActions, 1)
}

func createNominalDeployerTest() (*deployerTrait, *Environment) {
	trait := newDeployerTrait().(*deployerTrait)

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseRunning,
			},
		},
	}

	return trait, environment
}
