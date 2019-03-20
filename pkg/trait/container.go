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
	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
)

type containerTrait struct {
	BaseTrait     `property:",squash"`
	RequestCPU    string `property:"request-cpu"`
	RequestMemory string `property:"request-memory"`
	LimitCPU      string `property:"limit-cpu"`
	LimitMemory   string `property:"limit-memory"`
}

func newContainerTrait() *containerTrait {
	return &containerTrait{
		BaseTrait: newBaseTrait("container"),
	}
}

func (t *containerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *containerTrait) Apply(e *Environment) error {

	if e.Resources != nil {
		//
		// Add mounted volumes as resources
		//
		e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
			for i := 0; i < len(deployment.Spec.Template.Spec.Containers); i++ {
				t.configureResources(e, &deployment.Spec.Template.Spec.Containers[i])
			}
		})
		e.Resources.VisitKnativeService(func(service *serving.Service) {
			t.configureResources(e, &service.Spec.RunLatest.Configuration.RevisionTemplate.Spec.Container)
		})
	}

	return nil
}

func (t *containerTrait) configureResources(e *Environment, container *corev1.Container) error {

	//
	// Requests
	//

	if container.Resources.Requests == nil {
		container.Resources.Requests = make(corev1.ResourceList)
	}

	if t.RequestCPU != "" {
		v, err := resource.ParseQuantity(t.RequestCPU)
		if err != nil {
			return err
		}

		container.Resources.Requests[corev1.ResourceCPU] = v
	}
	if t.RequestMemory != "" {
		v, err := resource.ParseQuantity(t.RequestMemory)
		if err != nil {
			return err
		}

		container.Resources.Requests[corev1.ResourceMemory] = v
	}

	//
	// Limits
	//

	if container.Resources.Limits == nil {
		container.Resources.Limits = make(corev1.ResourceList)
	}

	if t.LimitCPU != "" {
		v, err := resource.ParseQuantity(t.LimitCPU)
		if err != nil {
			return err
		}

		container.Resources.Limits[corev1.ResourceCPU] = v
	}
	if t.LimitMemory != "" {
		v, err := resource.ParseQuantity(t.LimitMemory)
		if err != nil {
			return err
		}

		container.Resources.Limits[corev1.ResourceMemory] = v
	}

	return nil
}
