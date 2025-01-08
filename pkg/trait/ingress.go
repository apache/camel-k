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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	ingressTraitID    = "ingress"
	ingressTraitOrder = 2400

	defaultPath           = "/"
	defaultPathTypePrefix = networkingv1.PathTypePrefix
)

type ingressTrait struct {
	BaseTrait
	traitv1.IngressTrait `property:",squash"`
}

func newIngressTrait() Trait {
	return &ingressTrait{
		BaseTrait: NewBaseTrait(ingressTraitID, ingressTraitOrder),
	}
}

// IsAllowedInProfile overrides default.
func (t *ingressTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile.Equal(v1.TraitProfileKubernetes)
}

func (t *ingressTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationCondition(
			"Ingress",
			v1.IntegrationConditionExposureAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionIngressNotAvailableReason,
			"explicitly disabled",
		), nil
	}

	if ptr.Deref(t.Auto, true) {
		if e.Resources.GetUserServiceForIntegration(e.Integration) == nil {
			return false, nil, nil
		}
	}

	if t.Path != "" {
		m := "The path parameter is deprecated and may be removed in a future release. Use the paths parameter instead."
		t.L.Info(m)
		condition := NewIntegrationCondition(
			"Ingress",
			v1.IntegrationConditionTraitInfo,
			corev1.ConditionTrue,
			TraitConfigurationReason,
			m,
		)

		return true, condition, nil
	}

	return true, nil, nil
}

func (t *ingressTrait) Apply(e *Environment) error {
	service := e.Resources.GetUserServiceForIntegration(e.Integration)
	if service == nil {
		return errors.New("cannot apply ingress trait: no target service")
	}

	ingress := networkingv1.Ingress{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Ingress",
			APIVersion: networkingv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        service.Name,
			Namespace:   service.Namespace,
			Annotations: t.Annotations,
		},
		Spec: networkingv1.IngressSpec{
			Rules: []networkingv1.IngressRule{
				{
					Host: t.Host,
					IngressRuleValue: networkingv1.IngressRuleValue{
						HTTP: &networkingv1.HTTPIngressRuleValue{
							Paths: t.getPaths(service),
						},
					},
				},
			},
		},
	}
	if t.IngressClassName != "" {
		ingress.Spec.IngressClassName = &t.IngressClassName
	}

	if len(t.TLSHosts) > 0 && t.TLSSecretName != "" {
		ingress.Spec.TLS = []networkingv1.IngressTLS{
			{
				Hosts:      t.TLSHosts,
				SecretName: t.TLSSecretName,
			},
		}
	}

	e.Resources.Add(&ingress)

	message := fmt.Sprintf("%s(%s) -> %s(%s)", ingress.Name, t.Host, service.Name, "http")

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionExposureAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionIngressAvailableReason,
		message,
	)

	return nil
}

func (t *ingressTrait) getPaths(service *corev1.Service) []networkingv1.HTTPIngressPath {
	createIngressPath := func(path string) networkingv1.HTTPIngressPath {
		return networkingv1.HTTPIngressPath{
			Path:     path,
			PathType: t.getPathType(),
			Backend: networkingv1.IngressBackend{
				Service: &networkingv1.IngressServiceBackend{
					Name: service.Name,
					Port: networkingv1.ServiceBackendPort{
						Name: "http",
					},
				},
			},
		}
	}

	paths := []networkingv1.HTTPIngressPath{}
	if t.Path == "" && len(t.Paths) == 0 {
		paths = append(paths, createIngressPath(defaultPath))
	} else {
		if t.Path != "" {
			paths = append(paths, createIngressPath(t.Path))
		}
		for _, p := range t.Paths {
			paths = append(paths, createIngressPath(p))
		}
	}

	return paths
}

func (t *ingressTrait) getPathType() *networkingv1.PathType {
	if t.PathType == nil {
		return ptr.To(defaultPathTypePrefix)
	}

	return t.PathType
}
