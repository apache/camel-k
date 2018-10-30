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
	"github.com/apache/camel-k/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

type ingressTrait struct {
	BaseTrait `property:",squash"`
	Host      string `property:"host"`
}

func newIngressTrait() *ingressTrait {
	return &ingressTrait{
		BaseTrait: newBaseTrait("ingress"),
		Host:      "",
	}
}

func (e *ingressTrait) autoconfigure(environment *environment, resources *kubernetes.Collection) error {
	if e.Enabled == nil {
		hasService := e.getTargetService(environment, resources) != nil
		hasHost := e.Host != ""
		enabled := hasService && hasHost
		e.Enabled = &enabled
	}
	return nil
}

func (e *ingressTrait) customize(environment *environment, resources *kubernetes.Collection) error {
	if e.Host == "" {
		return errors.New("cannot apply ingress trait: no host defined")
	}
	service := e.getTargetService(environment, resources)
	if service == nil {
		return errors.New("cannot apply ingress trait: no target service")
	}

	resources.Add(e.getIngressFor(environment, service))
	return nil
}

func (*ingressTrait) getTargetService(e *environment, resources *kubernetes.Collection) (service *corev1.Service) {
	resources.VisitService(func(s *corev1.Service) {
		if s.ObjectMeta.Labels != nil {
			if intName, ok := s.ObjectMeta.Labels["camel.apache.org/integration"]; ok && intName == e.Integration.Name {
				service = s
			}
		}
	})
	return
}

func (e *ingressTrait) getIngressFor(env *environment, service *corev1.Service) *v1beta1.Ingress {
	ingress := v1beta1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		Spec: v1beta1.IngressSpec{
			Backend: &v1beta1.IngressBackend{
				ServiceName: service.Name,
				ServicePort: intstr.FromString("http"),
			},
			Rules: []v1beta1.IngressRule{
				{
					Host: e.Host,
				},
			},
		},
	}
	return &ingress
}
