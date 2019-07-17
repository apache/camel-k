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
	"github.com/apache/camel-k/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

type serviceTrait struct {
	BaseTrait `property:",squash"`
	Auto      *bool `property:"auto"`
}

const httpPortName = "http"

func newServiceTrait() *serviceTrait {
	return &serviceTrait{
		BaseTrait: newBaseTrait("service"),
	}
}

func (t *serviceTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		e.Integration.Status.SetCondition(
			v1alpha1.IntegrationConditionServiceAvailable,
			corev1.ConditionFalse,
			v1alpha1.IntegrationConditionServiceNotAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	if !e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		sources, err := kubernetes.ResolveIntegrationSources(t.ctx, t.client, e.Integration, e.Resources)
		if err != nil {
			e.Integration.Status.SetCondition(
				v1alpha1.IntegrationConditionServiceAvailable,
				corev1.ConditionFalse,
				v1alpha1.IntegrationConditionServiceNotAvailableReason,
				err.Error(),
			)

			return false, err
		}

		meta := metadata.ExtractAll(e.CamelCatalog, sources)
		if !meta.RequiresHTTPService {
			e.Integration.Status.SetCondition(
				v1alpha1.IntegrationConditionServiceAvailable,
				corev1.ConditionFalse,
				v1alpha1.IntegrationConditionServiceNotAvailableReason,
				"no http service required",
			)

			return false, nil
		}
	}

	return true, nil
}

func (t *serviceTrait) Apply(e *Environment) (err error) {
	svc := e.Resources.GetServiceForIntegration(e.Integration)
	if svc == nil {
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
				Ports: []corev1.ServicePort{},
				Selector: map[string]string{
					"camel.apache.org/integration": e.Integration.Name,
				},
			},
		}

		// add a new service if not already created
		e.Resources.Add(&svc)
	}

	return nil
}
