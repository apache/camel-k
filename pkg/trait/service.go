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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type serviceTrait struct {
	BaseTrait
	traitv1.ServiceTrait `property:",squash"`
}

const serviceTraitID = "service"

func newServiceTrait() Trait {
	return &serviceTrait{
		BaseTrait: NewBaseTrait(serviceTraitID, 1500),
	}
}

func (t *serviceTrait) Configure(e *Environment) (bool, error) {
	if e.Integration == nil || !pointer.BoolDeref(t.Enabled, true) {
		if e.Integration != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionServiceAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionServiceNotAvailableReason,
				"explicitly disabled",
			)
		}

		return false, nil
	}

	// in case the knative-service and service trait are enabled, the knative-service has priority
	// then this service is disabled
	if e.GetTrait(knativeServiceTraitID) != nil {
		knativeServiceTrait, _ := e.GetTrait(knativeServiceTraitID).(*knativeServiceTrait)
		if pointer.BoolDeref(knativeServiceTrait.Enabled, true) {
			return false, nil
		}
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if pointer.BoolDeref(t.Auto, true) {
		sources, err := kubernetes.ResolveIntegrationSources(e.Ctx, t.Client, e.Integration, e.Resources)
		if err != nil {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionServiceAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionServiceNotAvailableReason,
				err.Error(),
			)

			return false, err
		}

		meta, err := metadata.ExtractAll(e.CamelCatalog, sources)
		if err != nil {
			return false, err
		}
		if !meta.ExposesHTTPServices {
			e.Integration.Status.SetCondition(
				v1.IntegrationConditionServiceAvailable,
				corev1.ConditionFalse,
				v1.IntegrationConditionServiceNotAvailableReason,
				"no http service required",
			)

			return false, nil
		}
	}
	return true, nil
}

func (t *serviceTrait) Apply(e *Environment) error {
	svc := e.Resources.GetServiceForIntegration(e.Integration)
	// add a new service if not already created
	if svc == nil {
		svc = getServiceFor(e)

		var serviceType corev1.ServiceType
		if t.Type != nil {
			switch *t.Type {
			case traitv1.ServiceTypeClusterIP:
				serviceType = corev1.ServiceTypeClusterIP
			case traitv1.ServiceTypeNodePort:
				serviceType = corev1.ServiceTypeNodePort
			case traitv1.ServiceTypeLoadBalancer:
				serviceType = corev1.ServiceTypeLoadBalancer
			default:
				return fmt.Errorf("unsupported service type: %s", *t.Type)
			}
		} else if pointer.BoolDeref(t.NodePort, false) {
			t.L.ForIntegration(e.Integration).Infof("Integration %s/%s should no more use the flag node-port as it is deprecated, use type instead", e.Integration.Namespace, e.Integration.Name)
			serviceType = corev1.ServiceTypeNodePort
		}
		svc.Spec.Type = serviceType
	}
	e.Resources.Add(svc)
	return nil
}

func getServiceFor(e *Environment) *corev1.Service {
	return &corev1.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
			Labels: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
		},
		Spec: corev1.ServiceSpec{
			Ports: []corev1.ServicePort{},
			Selector: map[string]string{
				v1.IntegrationLabel: e.Integration.Name,
			},
		},
	}
}
