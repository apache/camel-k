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

	corev1 "k8s.io/api/core/v1"
	networking "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// The Ingress trait can be used to expose the service associated with the integration
// to the outside world with a Kubernetes Ingress.
//
// It's enabled by default whenever a Service is added to the integration (through the `service` trait).
//
// +camel-k:trait=ingress
type ingressTrait struct {
	BaseTrait `property:",squash"`
	// **Required**. To configure the host exposed by the ingress.
	Host string `property:"host" json:"host,omitempty"`
	// To automatically add an ingress whenever the integration uses a HTTP endpoint consumer.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}

func newIngressTrait() Trait {
	return &ingressTrait{
		BaseTrait: NewBaseTrait("ingress", 2400),
		Host:      "",
	}
}

// IsAllowedInProfile overrides default
func (t *ingressTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile == v1.TraitProfileKubernetes
}

func (t *ingressTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionExposureAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionIngressNotAvailableReason,
			"explicitly disabled",
		)
		return false, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if IsNilOrTrue(t.Auto) {
		hasService := e.Resources.GetUserServiceForIntegration(e.Integration) != nil
		hasHost := t.Host != ""
		enabled := hasService && hasHost

		if !enabled {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionExposureAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionIngressNotAvailableReason,
				"no host or service defined",
			)

			return false, nil
		}
	}

	if t.Host == "" {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionExposureAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionIngressNotAvailableReason,
			"no host defined",
		)

		return false, errors.New("cannot Apply ingress trait: no host defined")
	}

	return true, nil
}

func (t *ingressTrait) Apply(e *Environment) error {
	service := e.Resources.GetUserServiceForIntegration(e.Integration)
	if service == nil {
		return errors.New("cannot Apply ingress trait: no target service")
	}

	ingress := networking.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networking.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      service.Name,
			Namespace: service.Namespace,
		},
		Spec: networking.IngressSpec{
			DefaultBackend: &networking.IngressBackend{
				Service: &networking.IngressServiceBackend{
					Name: service.Name,
					Port: networking.ServiceBackendPort{
						Name: "http",
					},
				},
			},
			Rules: []networking.IngressRule{
				{
					Host: t.Host,
				},
			},
		},
	}

	e.Resources.Add(&ingress)

	message := fmt.Sprintf("%s(%s) -> %s(%s)",
		ingress.Name,
		t.Host,
		ingress.Spec.DefaultBackend.Service.Name,
		ingress.Spec.DefaultBackend.Service.Port.Name)

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionIngressAvailableReason,
		message,
	)

	return nil
}
