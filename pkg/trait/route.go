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
	"fmt"
	"reflect"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	routev1 "github.com/openshift/api/route/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// The Route trait can be used to configure the creation of OpenShift routes for the integration.
//
// +camel-k:trait=route
type routeTrait struct {
	BaseTrait `property:",squash"`
	// To configure the host exposed by the route.
	Host string `property:"host" json:"host,omitempty"`
	// The TLS termination type, like `edge`, `passthrough` or `reencrypt`.
	//
	// Refer to the OpenShift documentation for additional information.
	TLSTermination string `property:"tls-termination" json:"tlsTermination,omitempty"`
	// The TLS certificate contents.
	//
	// Refer to the OpenShift documentation for additional information.
	TLSCertificate string `property:"tls-certificate" json:"tlsCertificate,omitempty"`
	// The TLS certificate key contents.
	//
	// Refer to the OpenShift documentation for additional information.
	TLSKey string `property:"tls-key" json:"tlsKey,omitempty"`
	// The TLS cert authority certificate contents.
	//
	// Refer to the OpenShift documentation for additional information.
	TLSCACertificate string `property:"tls-ca-certificate" json:"tlsCACertificate,omitempty"`
	// The destination CA certificate provides the contents of the ca certificate of the final destination.  When using reencrypt
	// termination this file should be provided in order to have routers use it for health checks on the secure connection.
	// If this field is not specified, the router may provide its own destination CA and perform hostname validation using
	// the short service name (service.namespace.svc), which allows infrastructure generated certificates to automatically
	// verify.
	//
	// Refer to the OpenShift documentation for additional information.
	TLSDestinationCACertificate string `property:"tls-destination-ca-certificate" json:"tlsDestinationCACertificate,omitempty"`
	// To configure how to deal with insecure traffic, e.g. `Allow`, `Disable` or `Redirect` traffic.
	//
	// Refer to the OpenShift documentation for additional information.
	TLSInsecureEdgeTerminationPolicy string `property:"tls-insecure-edge-termination-policy" json:"tlsInsecureEdgeTerminationPolicy,omitempty"`

	service *corev1.Service
}

func newRouteTrait() Trait {
	return &routeTrait{
		BaseTrait: NewBaseTrait("route", 2200),
	}
}

// IsAllowedInProfile overrides default
func (t *routeTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile == v1.TraitProfileOpenShift
}

func (t *routeTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		if e.Integration != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionExposureAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionRouteNotAvailableReason,
				"explicitly disabled",
			)
		}

		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning) {
		return false, nil
	}

	t.service = e.Resources.GetUserServiceForIntegration(e.Integration)
	if t.service == nil {
		if e.Integration != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionExposureAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionRouteNotAvailableReason,
				"no target service found",
			)
		}

		return false, nil
	}

	return true, nil
}

func (t *routeTrait) Apply(e *Environment) error {
	servicePortName := defaultContainerPortName
	dt := e.Catalog.GetTrait(containerTraitID)
	if dt != nil {
		servicePortName = dt.(*containerTrait).ServicePortName
	}

	route := routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: routev1.GroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      t.service.Name,
			Namespace: t.service.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString(servicePortName),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: t.service.Name,
			},
			Host: t.Host,
			TLS:  t.getTLSConfig(),
		},
	}

	e.Resources.Add(&route)

	var message string

	if t.Host == "" {
		message = fmt.Sprintf("%s -> %s(%s)",
			route.Name,
			route.Spec.To.Name,
			route.Spec.Port.TargetPort.String())
	} else {
		message = fmt.Sprintf("%s(%s) -> %s(%s)",
			route.Name,
			t.Host,
			route.Spec.To.Name,
			route.Spec.Port.TargetPort.String())
	}

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionRouteAvailableReason,
		message,
	)

	return nil
}

func (t *routeTrait) getTLSConfig() *routev1.TLSConfig {
	config := routev1.TLSConfig{
		Termination:                   routev1.TLSTerminationType(t.TLSTermination),
		Certificate:                   t.TLSCertificate,
		Key:                           t.TLSKey,
		CACertificate:                 t.TLSCACertificate,
		DestinationCACertificate:      t.TLSDestinationCACertificate,
		InsecureEdgeTerminationPolicy: routev1.InsecureEdgeTerminationPolicyType(t.TLSInsecureEdgeTerminationPolicy),
	}

	if reflect.DeepEqual(config, routev1.TLSConfig{}) {
		return nil
	}

	return &config
}
