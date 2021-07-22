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
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestApplyConfigurationTraitWithProperties(t *testing.T) {
	configurationTrait, environment := createNominalConfigurationTest()
	configurationTrait.Properties = []string{"a=b", "c=d"}
	err := configurationTrait.Apply(environment)
	assert.Nil(t, err)

	userPropertiesCm := environment.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
		return cm.Labels["camel.apache.org/properties.type"] == "user"
	})
	assert.NotNil(t, userPropertiesCm)
	assert.Equal(t, map[string]string{
		"application.properties": "a=b\nc=d\n",
	}, userPropertiesCm.Data)
}

func createNominalConfigurationTest() (*configurationTrait, *Environment) {
	trait := newConfigurationTrait().(*configurationTrait)
	trait.Enabled = BoolP(true)
	trait.Client, _ = test.NewFakeClient(&appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "integration-name",
			Namespace: "namespace",
			Labels: map[string]string{
				v1.IntegrationLabel: "integration-name",
			},
		},
	})

	replicas := int32(3)

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Spec: v1.IntegrationSpec{
				Replicas: &replicas,
				Traits:   make(map[string]v1.TraitSpec),
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
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
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Profile: v1.TraitProfileKnative,
			},
		},
		Resources:             kubernetes.NewCollection(),
		ApplicationProperties: make(map[string]string),
	}
	environment.Platform.ResyncStatusFullConfig()

	return trait, environment
}
