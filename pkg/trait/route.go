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
	"github.com/apache/camel-k/pkg/util/kubernetes"
	routev1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type routeTrait struct {
	BaseTrait `property:",squash"`
}

func newRouteTrait() *routeTrait {
	return &routeTrait{
		BaseTrait: newBaseTrait("route"),
	}
}

func (e *routeTrait) autoconfigure(environment *environment, resources *kubernetes.Collection) error {
	if e.Enabled == nil {
		hasService := e.getTargetService(environment, resources) != nil
		e.Enabled = &hasService
	}
	return nil
}

func (e *routeTrait) customize(environment *environment, resources *kubernetes.Collection) error {
	service := e.getTargetService(environment, resources)
	if service != nil {
		resources.Add(e.getRouteFor(environment, service))
	}

	return nil
}

func (*routeTrait) getTargetService(e *environment, resources *kubernetes.Collection) (service *corev1.Service) {
	resources.VisitService(func(s *corev1.Service) {
		if s.ObjectMeta.Labels != nil {
			if intName, ok := s.ObjectMeta.Labels["camel.apache.org/integration"]; ok && intName == e.Integration.Name {
				service = s
			}
		}
	})
	return
}

func (*routeTrait) getRouteFor(e *environment, service *corev1.Service) *routev1.Route {
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
		},
	}
	return &route
}
