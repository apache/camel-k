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
	"sort"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util"
	"k8s.io/apimachinery/pkg/util/intstr"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

type probesTrait struct {
	BaseTrait `property:",squash"`

	BindHost string `property:"bind-host"`
	BindPort int    `property:"bind-port"`
	Path     string `property:"path"`
}

func newProbesTrait() *probesTrait {
	return &probesTrait{
		BaseTrait: newBaseTrait("probes"),
		BindHost:  "0.0.0.0",
		BindPort:  8081,
		Path:      "/health",
	}
}

func (t *probesTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && *t.Enabled {
		return e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitial) || e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
	}

	return false, nil
}

func (t *probesTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseInitial) {
		util.StringSliceUniqueAdd(&e.Integration.Status.Dependencies, "runtime:health")

		// sort the dependencies to get always the same list if they don't change
		sort.Strings(e.Integration.Status.Dependencies)
	}

	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying) {
		e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
			if len(deployment.Spec.Template.Spec.Containers) != 1 {
				return
			}

			deployment.Spec.Template.Spec.Containers[0].LivenessProbe = t.newProbe()
			deployment.Spec.Template.Spec.Containers[0].ReadinessProbe = t.newProbe()
		})

		e.Resources.VisitKnativeService(func(service *serving.Service) {
			service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.LivenessProbe = t.newProbe()
			service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container.ReadinessProbe = t.newProbe()
		})
	}

	return nil
}

func (t *probesTrait) newProbe() *corev1.Probe {
	return &corev1.Probe{
		Handler: corev1.Handler{
			HTTPGet: &corev1.HTTPGetAction{
				Port: intstr.FromInt(t.BindPort),
				Path: t.Path,
			},
		},
	}
}
