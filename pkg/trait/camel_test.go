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

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestConfigureEnabledCamelTraitSucceeds(t *testing.T) {
	trait, environment := createNominalCamelTest()

	configured, err := trait.Configure(environment)
	assert.Nil(t, err)
	assert.True(t, configured)
}

func TestConfigureDisabledCamelTraitFails(t *testing.T) {
	trait, environment := createNominalCamelTest()
	trait.Enabled = BoolP(false)

	configured, err := trait.Configure(environment)
	assert.NotNil(t, err)
	assert.False(t, configured)
}

func TestApplyCamelTraitSucceeds(t *testing.T) {
	trait, environment := createNominalCamelTest()

	err := trait.Apply(environment)
	assert.Nil(t, err)
	assert.Equal(t, "0.0.1", environment.RuntimeVersion)
	assert.Equal(t, "0.0.1", environment.Integration.Status.RuntimeVersion)
	assert.Equal(t, "0.0.1", environment.IntegrationKit.Status.RuntimeVersion)
}

func TestApplyCamelTraitWithoutEnvironmentCatalogAndUnmatchableVersionFails(t *testing.T) {
	trait, environment := createNominalCamelTest()
	environment.CamelCatalog = nil
	environment.Integration.Status.RuntimeVersion = "Unmatchable version"
	environment.Integration.Status.RuntimeProvider = v1.RuntimeProviderQuarkus

	err := trait.Apply(environment)
	assert.NotNil(t, err)
	assert.Equal(t, "unable to find catalog matching version requirement: runtime=Unmatchable version, provider=quarkus", err.Error())
}

func createNominalCamelTest() (*camelTrait, *Environment) {
	client, _ := test.NewFakeClient()

	trait := newCamelTrait().(*camelTrait)
	trait.Enabled = BoolP(true)

	environment := &Environment{
		CamelCatalog: &camel.RuntimeCatalog{
			CamelCatalogSpec: v1.CamelCatalogSpec{
				Runtime: v1.RuntimeSpec{
					Version:  "0.0.1",
					Provider: v1.RuntimeProviderQuarkus,
				},
			},
		},
		Catalog: NewEnvironmentTestCatalog(),
		Ctx:     context.TODO(),
		Client:  client,
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
			Spec: v1.IntegrationSpec{
				Traits: make(map[string]v1.TraitSpec),
			},
			Status: v1.IntegrationStatus{
				RuntimeVersion: "0.0.1",
				Phase:          v1.IntegrationPhaseDeploying,
			},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "namespace",
			},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}

	return trait, environment
}

func TestApplyCamelTraitWithProperties(t *testing.T) {
	camelTrait, environment := createNominalCamelTest()
	camelTrait.Properties = []string{"a=b", "c=d"}
	err := camelTrait.Apply(environment)
	assert.Nil(t, err)

	userPropertiesCm := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] == "user"
	})
	assert.NotNil(t, userPropertiesCm)
	assert.Equal(t, map[string]string{
		"application.properties": "a=b\nc=d\n",
	}, userPropertiesCm.Data)
}
