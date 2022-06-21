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

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

type podTrait struct {
	BaseTrait
	v1.PodTrait `property:",squash"`
}

func newPodTrait() Trait {
	return &podTrait{
		BaseTrait: NewBaseTrait("pod", 1800),
	}
}

func (t *podTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if e.Integration != nil && e.Integration.Spec.PodTemplate == nil {
		return false, nil
	}

	return e.IntegrationInRunningPhases(), nil
}

func (t *podTrait) Apply(e *Environment) error {
	changes := e.Integration.Spec.PodTemplate.Spec
	var patchedPodSpec *corev1.PodSpec
	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		return fmt.Errorf("unable to determine the controller strategy")
	}
	switch strategy {
	case ControllerStrategyCronJob:
		e.Resources.VisitCronJob(func(c *v1beta1.CronJob) {
			if c.Name == e.Integration.Name {
				if patchedPodSpec, err = t.applyChangesTo(&c.Spec.JobTemplate.Spec.Template.Spec, changes); err == nil {
					c.Spec.JobTemplate.Spec.Template.Spec = *patchedPodSpec
				}
			}
		})

	case ControllerStrategyDeployment:
		e.Resources.VisitDeployment(func(d *appsv1.Deployment) {
			if d.Name == e.Integration.Name {
				if patchedPodSpec, err = t.applyChangesTo(&d.Spec.Template.Spec, changes); err == nil {
					d.Spec.Template.Spec = *patchedPodSpec
				}
			}
		})

	case ControllerStrategyKnativeService:
		e.Resources.VisitKnativeService(func(s *serving.Service) {
			if s.Name == e.Integration.Name {
				if patchedPodSpec, err = t.applyChangesTo(&s.Spec.Template.Spec.PodSpec, changes); err == nil {
					s.Spec.Template.Spec.PodSpec = *patchedPodSpec
				}
			}
		})

	}
	if err != nil {
		return err
	}
	return nil
}

func (t *podTrait) applyChangesTo(podSpec *corev1.PodSpec, changes v1.PodSpec) (patchedPodSpec *corev1.PodSpec, err error) {
	patch, err := json.Marshal(changes)
	if err != nil {
		return
	}

	sourceJSON, err := json.Marshal(podSpec)
	if err != nil {
		return
	}

	patched, err := strategicpatch.StrategicMergePatch(sourceJSON, patch, corev1.PodSpec{})
	if err != nil {
		return
	}

	err = json.Unmarshal(patched, &patchedPodSpec)
	return
}
