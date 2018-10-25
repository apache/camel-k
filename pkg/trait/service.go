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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var webComponents = map[string]bool{
	"camel:servlet":     true,
	"camel:undertow":    true,
	"camel:jetty":       true,
	"camel:netty-http":  true,
	"camel:netty4-http": true,
	// TODO find a better way to discover need for exposure
	// maybe using the resolved classpath of the context instead of the requested dependencies
}

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

func (s *serviceTrait) autoconfigure(environment *environment, resources *kubernetes.Collection) error {
	if s.Enabled == nil {
		required := s.requiresService(environment)
		s.Enabled = &required
	}
	return nil
}

func (s *serviceTrait) customize(environment *environment, resources *kubernetes.Collection) (err error) {
	var svc *corev1.Service
	if svc, err = s.getServiceFor(environment); err != nil {
		return err
	}
	resources.Add(svc)
	return nil
}

func (s *serviceTrait) getServiceFor(e *environment) (*corev1.Service, error) {
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

	return &svc, nil
}

func (*serviceTrait) requiresService(environment *environment) bool {
	for _, dep := range environment.Integration.Spec.Dependencies {
		if decision, present := webComponents[dep]; present {
			return decision
		}
	}
	return false
}
