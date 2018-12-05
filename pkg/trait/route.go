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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type routeTrait struct {
	BaseTrait `property:",squash"`
	Host      string `property:"host"`
}

func newRouteTrait() *routeTrait {
	return &routeTrait{
		BaseTrait: newBaseTrait("route"),
	}
}

func (r *routeTrait) appliesTo(e *Environment) bool {
	return e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (r *routeTrait) autoconfigure(e *Environment) error {
	if r.Enabled == nil {
		hasService := r.getTargetService(e) != nil
		r.Enabled = &hasService
	}
	return nil
}

func (r *routeTrait) apply(e *Environment) error {
	if e.Integration == nil || e.Integration.Status.Phase != v1alpha1.IntegrationPhaseDeploying {
		return nil
	}

	service := r.getTargetService(e)
	if service == nil {
		return errors.New("cannot apply route trait: no target service")
	}

	e.Resources.Add(r.getRouteFor(service))
	return nil
}

func (*routeTrait) getTargetService(e *Environment) (service *corev1.Service) {
	e.Resources.VisitService(func(s *corev1.Service) {
		if s.ObjectMeta.Labels != nil {
			if intName, ok := s.ObjectMeta.Labels["camel.apache.org/integration"]; ok && intName == e.Integration.Name {
				service = s
			}
		}
	})
	return
}

func (r *routeTrait) getRouteFor(service *corev1.Service) *routev1.Route {
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
			Host: r.Host,
		},
	}
	return &route
}
