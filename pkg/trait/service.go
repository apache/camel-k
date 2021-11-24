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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// The Service trait exposes the integration with a Service resource so that it can be accessed by other applications
// (or integrations) in the same namespace.
//
// It's enabled by default if the integration depends on a Camel component that can expose a HTTP endpoint.
//
// +camel-k:trait=service.
type serviceTrait struct {
	BaseTrait `property:",squash"`
	// To automatically detect from the code if a Service needs to be created.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Enable Service to be exposed as NodePort
	NodePort *bool `property:"node-port" json:"nodePort,omitempty"`
}

const serviceTraitID = "service"

func newServiceTrait() Trait {
	return &serviceTrait{
		BaseTrait: NewBaseTrait(serviceTraitID, 1500),
	}
}

// IsAllowedInProfile overrides default.
func (t *serviceTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile == v1.TraitProfileKubernetes ||
		profile == v1.TraitProfileOpenShift
}

func (t *serviceTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionServiceNotAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil
	}

	if IsNilOrTrue(t.Auto) {
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

		meta := metadata.ExtractAll(e.CamelCatalog, sources)
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

		if IsNilOrTrue(t.NodePort) {
			svc.Spec.Type = corev1.ServiceTypeNodePort
		}
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
