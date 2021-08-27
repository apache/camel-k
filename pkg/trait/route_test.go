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

	"github.com/rs/xid"
	"github.com/stretchr/testify/assert"

	routev1 "github.com/openshift/api/route/v1"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/test"
)

func createTestRouteEnvironment(t *testing.T, name string) *Environment {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

	res := &Environment{
		CamelCatalog: catalog,
		Catalog:      NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: "test-ns",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
			Spec: v1.IntegrationSpec{},
		},
		IntegrationKit: &v1.IntegrationKit{
			Status: v1.IntegrationKitStatus{
				Phase: v1.IntegrationKitPhaseReady,
			},
		},
		Platform: &v1.IntegrationPlatform{
			Spec: v1.IntegrationPlatformSpec{
				Cluster: v1.IntegrationPlatformClusterOpenShift,
				Build: v1.IntegrationPlatformBuildSpec{
					PublishStrategy: v1.IntegrationPlatformBuildPublishStrategyS2I,
					Registry:        v1.IntegrationPlatformRegistrySpec{Address: "registry"},
				},
			},
		},
		EnvVars:        make([]corev1.EnvVar, 0),
		ExecutedTraits: make([]Trait, 0),
		Resources: kubernetes.NewCollection(
			&corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: "test-ns",
					Labels: map[string]string{
						v1.IntegrationLabel:             name,
						"camel.apache.org/service.type": v1.ServiceTypeUser,
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{},
					Selector: map[string]string{
						v1.IntegrationLabel: name,
					},
				},
			},
		),
	}
	res.Platform.ResyncStatusFullConfig()
	return res
}

func TestRoute_Default(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("container"))
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.Nil(t, route.Spec.TLS)
	assert.NotNil(t, route.Spec.Port)
	assert.Equal(t, defaultContainerPortName, route.Spec.Port.TargetPort.StrVal)
}

func TestRoute_Disabled(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	environment.Integration.Spec.Traits = map[string]v1.TraitSpec{
		"route": test.TraitSpecFromMap(t, map[string]interface{}{
			"enabled": false,
		}),
	}

	traitsCatalog := environment.Catalog
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.Nil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.Nil(t, route)
}

func TestRoute_Configure_IntegrationKitOnly(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	environment.Integration = nil

	routeTrait := newRouteTrait().(*routeTrait)
	enabled := false
	routeTrait.Enabled = &enabled

	result, err := routeTrait.Configure(environment)
	assert.False(t, result)
	assert.Nil(t, err)
}

func TestRoute_TLS(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	traitsCatalog := environment.Catalog

	environment.Integration.Spec.Traits = map[string]v1.TraitSpec{
		"route": test.TraitSpecFromMap(t, map[string]interface{}{
			"tlsTermination": string(routev1.TLSTerminationEdge),
		}),
	}

	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.TLS)
	assert.Equal(t, routev1.TLSTerminationEdge, route.Spec.TLS.Termination)
}

func TestRoute_WithCustomServicePort(t *testing.T) {
	name := xid.New().String()
	environment := createTestRouteEnvironment(t, name)
	environment.Integration.Spec.Traits = map[string]v1.TraitSpec{
		containerTraitID: test.TraitSpecFromMap(t, map[string]interface{}{
			"servicePortName": "my-port",
		}),
	}

	traitsCatalog := environment.Catalog
	err := traitsCatalog.apply(environment)

	assert.Nil(t, err)
	assert.NotEmpty(t, environment.ExecutedTraits)
	assert.NotNil(t, environment.GetTrait("container"))
	assert.NotNil(t, environment.GetTrait("route"))

	route := environment.Resources.GetRoute(func(r *routev1.Route) bool {
		return r.ObjectMeta.Name == name
	})

	assert.NotNil(t, route)
	assert.NotNil(t, route.Spec.Port)

	trait := test.TraitSpecToMap(t, environment.Integration.Spec.Traits[containerTraitID])
	assert.Equal(
		t,
		trait["servicePortName"],
		route.Spec.Port.TargetPort.StrVal,
	)
}
