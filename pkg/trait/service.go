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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type serviceTrait struct {
	BaseTrait `property:",squash"`

	Port int `property:"port"`
}

func newServiceTrait() *serviceTrait {
	return &serviceTrait{
		BaseTrait: newBaseTrait("service"),
		Port:      8080,
	}
}

func (s *serviceTrait) appliesTo(e *Environment) bool {
	return e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (s *serviceTrait) autoconfigure(e *Environment) error {
	if s.Enabled == nil {
		hasDeployment := false
		e.Resources.VisitDeployment(func(s *v1.Deployment) {
			hasDeployment = true
		})
		if hasDeployment {
			meta := metadata.ExtractAll(e.Integration.Spec.Sources)
			required := meta.RequiresHTTPService
			s.Enabled = &required
		} else {
			enabled := false
			s.Enabled = &enabled
		}
	}
	return nil
}

func (s *serviceTrait) apply(e *Environment) (err error) {
	svc := s.getServiceFor(e)
	e.Resources.Add(svc)
	return nil
}

func (s *serviceTrait) getServiceFor(e *Environment) *corev1.Service {
	svc := corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{
				{
					Name:       "http",
					Port:       80,
					Protocol:   corev1.ProtocolTCP,
					TargetPort: intstr.FromInt(s.Port),
				},
			},
			Selector: map[string]string{
				"camel.apache.org/integration": e.Integration.Name,
			},
		},
	}

	return &svc
}
