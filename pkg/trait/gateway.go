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
	"errors"
	"fmt"
	"strconv"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
	gwv1 "sigs.k8s.io/gateway-api/apis/v1"
)

const (
	gatewayTraitID    = "gateway"
	gatewayTraitOrder = 2420

	gatewayDefaultListener = "8080;HTTP"
)

type gatewayTrait struct {
	BaseTrait
	traitv1.GatewayTrait `property:",squash"`
}

func newGatewayTrait() Trait {
	return &gatewayTrait{
		BaseTrait: NewBaseTrait(gatewayTraitID, gatewayTraitOrder),
	}
}

func (t *gatewayTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if e.Resources.GetUserServiceForIntegration(e.Integration) == nil {
		return false, NewIntegrationCondition(
			"Gateway",
			v1.IntegrationConditionServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionServiceNotAvailableReason,
			"No service available. Skipping the trait execution",
		), nil
	}

	return true, nil, nil
}

func (t *gatewayTrait) Apply(e *Environment) error {
	service := e.Resources.GetUserServiceForIntegration(e.Integration)
	gwName := e.Integration.GetName()

	gw, err := buildGateway(gwName, e.Integration.GetNamespace(), t.ClassName, t.getListeners())
	if err != nil {
		return err
	}
	e.Resources.Add(gw)
	servicePorts := extractPorts(service.Spec.Ports)
	route := buildHTTPRoute(gwName, gw.GetName(), service.GetName(), gw.GetNamespace(), servicePorts)
	e.Resources.Add(route)

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionTrue,
		"GatewayAvailable",
		"Service is exposed via a Gateway and HTTPRoute named "+gwName,
	)

	return nil
}

func (t *gatewayTrait) getListeners() []string {
	if t.Listeners != nil {
		return t.Listeners
	}

	return []string{gatewayDefaultListener}
}

// buildGateway provides the gateway with the associated listeners.
func buildGateway(name, namespace, className string, listeners []string) (*gwv1.Gateway, error) {
	gwListeners := make([]gwv1.Listener, 0, len(listeners))

	for _, l := range listeners {
		parts := strings.Split(l, ";")
		if len(parts) != 2 {
			return nil, errors.New("could not parse gateway listener " + l)
		}

		port32, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			return nil, errors.New("could not parse gateway port " + parts[0])
		}
		port := int32(port32)
		protocol := strings.ToUpper(parts[1])
		if !isSupported(protocol) {
			return nil, errors.New("protocol gateway " + protocol + " is not yet supported: open change request issue to project tracking")
		}

		listenerName := fmt.Sprintf("%s-%d", name, port)
		gwListeners = append(gwListeners, gwv1.Listener{
			Name:     gwv1.SectionName(listenerName),
			Port:     gwv1.PortNumber(port),
			Protocol: gwv1.ProtocolType(protocol),
			AllowedRoutes: &gwv1.AllowedRoutes{
				Namespaces: &gwv1.RouteNamespaces{
					From: ptr.To(gwv1.NamespacesFromSame),
				},
			},
		})
	}

	return &gwv1.Gateway{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gwv1.SchemeGroupVersion.String(),
			Kind:       "Gateway",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
		},
		Spec: gwv1.GatewaySpec{
			GatewayClassName: gwv1.ObjectName(className),
			Listeners:        gwListeners,
		},
	}, nil
}

func isSupported(protocol string) bool {
	return protocol == "HTTP" || protocol == "HTTPS"
}

// buildHTTPRoute provides the most basic gateway builder method.
func buildHTTPRoute(routeName, gatewayName, serviceName, namespace string, servicePorts []int32) *gwv1.HTTPRoute {
	rules := make([]gwv1.HTTPRouteRule, 0, len(servicePorts))

	for _, p := range servicePorts {
		port := gwv1.PortNumber(p)
		rule := gwv1.HTTPRouteRule{
			BackendRefs: []gwv1.HTTPBackendRef{
				{
					BackendRef: gwv1.BackendRef{
						BackendObjectReference: gwv1.BackendObjectReference{
							Name: gwv1.ObjectName(serviceName),
							Port: ptr.To(port),
						},
					},
				},
			},
		}

		rules = append(rules, rule)
	}

	return &gwv1.HTTPRoute{
		TypeMeta: metav1.TypeMeta{
			APIVersion: gwv1.SchemeGroupVersion.String(),
			Kind:       "HTTPRoute",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      routeName,
			Namespace: namespace,
		},
		Spec: gwv1.HTTPRouteSpec{
			CommonRouteSpec: gwv1.CommonRouteSpec{
				ParentRefs: []gwv1.ParentReference{
					{
						Name: gwv1.ObjectName(gatewayName),
					},
				},
			},
			Rules: rules,
		},
	}
}

func extractPorts(ports []corev1.ServicePort) []int32 {
	result := make([]int32, 0, len(ports))
	for _, p := range ports {
		result = append(result, p.Port)
	}

	return result
}
