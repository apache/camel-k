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
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

func TestConfigureGatewayTraitDoesSucceed(t *testing.T) {
	gwTrait, environment := createNominalGatewayTest()
	gwTrait.ClassName = "my-gw-class"
	configured, condition, err := gwTrait.Configure(environment)

	require.NoError(t, err)
	assert.True(t, configured)
	assert.Nil(t, condition)
	err = gwTrait.Apply(environment)
	require.NoError(t, err)

	// Assert gateway resource
	var gateway gwv1.Gateway
	environment.Resources.Visit(func(o runtime.Object) {
		if conv, ok := o.(*gwv1.Gateway); ok {
			gateway = *conv
			return
		}
	})
	assert.NotNil(t, gateway, "Could not find any generated Gateway")
	assert.Equal(t, "integration-name", gateway.Name)
	assert.Equal(t, gwv1.ObjectName("my-gw-class"), gateway.Spec.GatewayClassName)
	assert.Len(t, gateway.Spec.Listeners, 1)
	assert.Equal(t, gwv1.ProtocolType("HTTP"), gateway.Spec.Listeners[0].Protocol)
	assert.Equal(t, gwv1.PortNumber(8080), gateway.Spec.Listeners[0].Port)

	// Assert HTTPRoute resource
	var httpRoute gwv1.HTTPRoute
	environment.Resources.Visit(func(o runtime.Object) {
		if conv, ok := o.(*gwv1.HTTPRoute); ok {
			httpRoute = *conv
			return
		}
	})
	assert.NotNil(t, httpRoute, "Could not find any generated HTTPRoute")
	assert.Len(t, httpRoute.Spec.ParentRefs, 1)
	assert.Equal(t, gwv1.ObjectName(gateway.Name), httpRoute.Spec.ParentRefs[0].Name)
	assert.Len(t, httpRoute.Spec.Rules, 2)
	assert.Contains(t, httpRoute.Spec.Rules,
		gwv1.HTTPRouteRule{
			BackendRefs: []gwv1.HTTPBackendRef{
				{BackendRef: gwv1.BackendRef{BackendObjectReference: gwv1.BackendObjectReference{
					Name: "service-name", Port: ptr.To(gwv1.PortNumber(1234)),
				}}},
			},
		},
	)
	assert.Contains(t, httpRoute.Spec.Rules,
		gwv1.HTTPRouteRule{
			BackendRefs: []gwv1.HTTPBackendRef{
				{BackendRef: gwv1.BackendRef{BackendObjectReference: gwv1.BackendObjectReference{
					Name: "service-name", Port: ptr.To(gwv1.PortNumber(5678)),
				}}},
			},
		},
	)

	// Verify Integration condition as well
	assert.NotNil(t, environment.Integration.Status.GetCondition(v1.IntegrationConditionExposureAvailable))
	assert.Equal(t, corev1.ConditionTrue, environment.Integration.Status.GetCondition(v1.IntegrationConditionExposureAvailable).Status)
	assert.Equal(t, "Service is exposed via a Gateway and HTTPRoute named integration-name",
		environment.Integration.Status.GetCondition(v1.IntegrationConditionExposureAvailable).Message)
}

func TestConfigureGatewayTraitMissingService(t *testing.T) {
	gwTrait, environment := createNominalGatewayTest()
	gwTrait.ClassName = "my-gw-class"
	environment.Resources.Remove(func(o runtime.Object) bool {
		if _, ok := o.(*corev1.Service); ok {
			return true
		}

		return false
	})
	configured, condition, err := gwTrait.Configure(environment)

	require.NoError(t, err)
	assert.False(t, configured)
	assert.NotNil(t, condition)
	assert.Contains(t, condition.message, "No service available")
}

func createNominalGatewayTest() (*gatewayTrait, *Environment) {
	trait, _ := newGatewayTrait().(*gatewayTrait)
	trait.Enabled = ptr.To(true)

	environment := &Environment{
		Catalog: NewCatalog(nil),
		Integration: &v1.Integration{
			ObjectMeta: metav1.ObjectMeta{
				Name: "integration-name",
			},
			Status: v1.IntegrationStatus{
				Phase: v1.IntegrationPhaseDeploying,
			},
		},
		Resources: kubernetes.NewCollection(
			&corev1.Service{
				TypeMeta: metav1.TypeMeta{
					Kind:       "Service",
					APIVersion: "v1",
				},
				ObjectMeta: metav1.ObjectMeta{
					Name:      "service-name",
					Namespace: "namespace",
					Labels: map[string]string{
						v1.IntegrationLabel:             "integration-name",
						"camel.apache.org/service.type": v1.ServiceTypeUser,
					},
				},
				Spec: corev1.ServiceSpec{
					Ports: []corev1.ServicePort{
						{
							Port: 1234,
						},
						{
							Port: 5678,
						},
					},
					Selector: map[string]string{
						v1.IntegrationLabel: "integration-name",
					},
				},
			},
		),
	}

	return trait, environment
}
