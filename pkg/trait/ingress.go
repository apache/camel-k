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

func (*ingressTrait) appliesTo(e *Environment) bool {
	return e.Integration != nil && e.Integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (i *ingressTrait) autoconfigure(e *Environment) error {
	if i.Enabled == nil {
		hasService := i.getTargetService(e) != nil
		hasHost := i.Host != ""
		enabled := hasService && hasHost
		i.Enabled = &enabled
	}
	return nil
}

func (i *ingressTrait) apply(e *Environment) error {
	if i.Host == "" {
		return errors.New("cannot apply ingress trait: no host defined")
	}
	service := i.getTargetService(e)
	if service == nil {
		return errors.New("cannot apply ingress trait: no target service")
	}

	e.Resources.Add(i.getIngressFor(e, service))
	return nil
}

func (*ingressTrait) getTargetService(e *Environment) (service *corev1.Service) {
	e.Resources.VisitService(func(s *corev1.Service) {
		if s.ObjectMeta.Labels != nil {
			if intName, ok := s.ObjectMeta.Labels["camel.apache.org/integration"]; ok && intName == e.Integration.Name {
				service = s
			}
		}
	})
	return
}

func (i *ingressTrait) getIngressFor(env *Environment, service *corev1.Service) *v1beta1.Ingress {
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
					Host: i.Host,
				},
			},
		},
	}
	return &ingress
}
