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
	Auto      *bool  `property:"auto"`
}

func newIngressTrait() *ingressTrait {
	return &ingressTrait{
		BaseTrait: newBaseTrait("ingress"),
		Host:      "",
	}
}

func (t *ingressTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		hasService := t.getTargetService(e) != nil
		hasHost := t.Host != ""
		enabled := hasService && hasHost

		if !enabled {
			return false, nil
		}
	}

	if t.Host == "" {
		return false, errors.New("cannot Apply ingress trait: no host defined")
	}

	return true, nil
}

func (t *ingressTrait) Apply(e *Environment) error {
	if t.Host == "" {
		return errors.New("cannot Apply ingress trait: no host defined")
	}
	service := t.getTargetService(e)
	if service == nil {
		return errors.New("cannot Apply ingress trait: no target service")
	}

	e.Resources.Add(t.getIngressFor(service))
	return nil
}

func (t *ingressTrait) getTargetService(e *Environment) (service *corev1.Service) {
	e.Resources.VisitService(func(s *corev1.Service) {
		if s.ObjectMeta.Labels != nil {
			if intName, ok := s.ObjectMeta.Labels["camel.apache.org/integration"]; ok && intName == e.Integration.Name {
				if s.ObjectMeta.Labels["camel.apache.org/service.type"] == "user" {
					// We should build an ingress only on top of the user service (e.g. not if the service contains only prometheus)
					service = s
				}
			}
		}
	})
	return
}

func (t *ingressTrait) getIngressFor(service *corev1.Service) *v1beta1.Ingress {
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
					Host: t.Host,
				},
			},
		},
	}
	return &ingress
}
