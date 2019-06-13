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
	"reflect"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type routeTrait struct {
	BaseTrait                        `property:",squash"`
	Auto                             *bool  `property:"auto"`
	Host                             string `property:"host"`
	TLSTermination                   string `property:"tls-termination"`
	TLSCertificate                   string `property:"tls-certificate"`
	TLSKey                           string `property:"tls-key"`
	TLSCACertificate                 string `property:"tls-ca-certificate"`
	TLSDestinationCACertificate      string `property:"tls-destination-ca-certificate"`
	TLSInsecureEdgeTerminationPolicy string `property:"tls-insecure-edge-termination-policy"`
	service                          *corev1.Service
}

func newRouteTrait() *routeTrait {
	return &routeTrait{
		BaseTrait: newBaseTrait("route"),
	}
}

func (t *routeTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		t.service = t.getTargetService(e)
		if t.service == nil {
			return false, nil
		}
	}

	if t.service == nil {
		return false, errors.New("cannot apply route trait: no target service")
	}

	return true, nil
}

func (t *routeTrait) Apply(e *Environment) error {
	if e.Integration == nil || e.Integration.Status.Phase != v1alpha1.IntegrationPhaseDeploying {
		return nil
	}

	e.Resources.Add(t.getRouteFor(t.service))
	return nil
}

func (t *routeTrait) getTargetService(e *Environment) (service *corev1.Service) {
	e.Resources.VisitService(func(s *corev1.Service) {
		if s.ObjectMeta.Labels != nil {
			if intName, ok := s.ObjectMeta.Labels["camel.apache.org/integration"]; ok && intName == e.Integration.Name {
				if s.ObjectMeta.Labels["camel.apache.org/service.type"] == "user" {
					// We should build a route only on top of the user service (e.g. not if the service contains only prometheus)
					service = s
				}
			}
		}
	})
	return
}

func (t *routeTrait) getRouteFor(service *corev1.Service) *routev1.Route {
	route := routev1.Route{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Route",
			APIVersion: routev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		Spec: routev1.RouteSpec{
			Port: &routev1.RoutePort{
				TargetPort: intstr.FromString("http"),
			},
			To: routev1.RouteTargetReference{
				Kind: "Service",
				Name: service.Name,
			},
			Host: t.Host,
			TLS:  t.getTLSConfig(),
		},
	}
	return &route
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
