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

	"github.com/scylladb/go-set/strset"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"

	"github.com/stretchr/testify/assert"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createTestRouteEnvironment(t *testing.T) *Environment {
	catalog, err := test.DefaultCatalog()
	assert.Nil(t, err)

	return &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(context.TODO(), nil),
		Integration: &v1alpha1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-i",
				Namespace: "test-ns",
			},
			Status: v1alpha1.IntegrationStatus{
				Phase: v1alpha1.IntegrationPhaseDeploying,
			},
			Spec: v1alpha1.IntegrationSpec{},
		},
		IntegrationKit: &v1alpha1.IntegrationKit{
			Status: v1alpha1.IntegrationKitStatus{
				Phase: v1alpha1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1alpha1.IntegrationPlatform{
			Spec: v1alpha1.IntegrationPlatformSpec{
				Cluster: v1alpha1.IntegrationPlatformClusterOpenShift,
				Build: v1alpha1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1alpha1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1alpha1.IntegrationPlatformRegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Classpath:      strset.New(),
		Resources: kubernetes.NewCollection(&corev1.Service{
			TypeMeta: metav1.TypeMeta{
				Kind:       "Service",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "test-i",
				Namespace: "test-ns",
				Labels: map[string]string{
					"camel.apache.org/integration": "test-i",
					"camel.apache.org/service.type": "user",
				},
			},
			Spec: corev1.ServiceSpec{
				Ports: []corev1.ServicePort{},
				Selector: map[string]string{
					"camel.apache.org/integration": "test-i",
				},
			},
		}),
	}
}

func TestRoute_Default(t *testing.T) {
	environment := createTestRouteEnvironment(t)
	traitsCatalog := environment.Catalog

	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(ID("route")))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == "test-i"
	})

	assert.NotNil(t, route)
	assert.Nil(t, route.Spec.TLS)
}

func TestRoute_TLS(t *testing.T) {
	environment := createTestRouteEnvironment(t)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = map[string]v1alpha1.TraitSpec{
		"route": {
			Configuration: map[string]string{
				"tls-termination": string(routev1.TLSTerminationEdge),
			},
		},
	}

	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait(ID("route")))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == "test-i"
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.TLS)
	assert.Equal(t, routev1.TLSTerminationEdge, route.Spec.TLS.Termination)
}
