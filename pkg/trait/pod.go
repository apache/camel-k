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

	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/util/json"
	"k8s.io/apimachinery/pkg/util/strategicpatch"
	"k8s.io/utils/ptr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

const podTraitOrder = 1800

type podTrait struct {
	BaseTrait
	traitv1.PodTrait `property:",squash"`
}

func newPodTrait() Trait {
	return &podTrait{
		BaseTrait: NewBaseTrait("pod", podTraitOrder),
	}
}

func (t *podTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationConditionUserDisabled("Pod"), nil
	}
	if e.Integration.Spec.PodTemplate == nil {
		return false, nil, nil
	}

	condition := NewIntegrationCondition(
		"Pod",
		v1.IntegrationConditionTraitInfo,
		corev1.ConditionTrue,
		TraitConfigurationReason,
		"Pod trait is deprecated in favour of InitContainers. It may be removed in future version.",
	)

	return e.IntegrationInRunningPhases(), condition, nil
}

func (t *podTrait) Apply(e *Environment) error {
	changes := e.Integration.Spec.PodTemplate.Spec
	var patchedPodSpec *corev1.PodSpec
	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		return errors.New("unable to determine the controller strategy")
	}
	switch strategy {
	case ControllerStrategyCronJob:
		e.Resources.VisitCronJob(func(c *batchv1.CronJob) {
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

func (t *podTrait) applyChangesTo(podSpec *corev1.PodSpec, changes v1.PodSpec) (*corev1.PodSpec, error) {
	patch, err := json.Marshal(changes)
	if err != nil {
		return nil, err
	}

	sourceJSON, err := json.Marshal(podSpec)
	if err != nil {
		return nil, err
	}

	patched, err := strategicpatch.StrategicMergePatch(sourceJSON, patch, corev1.PodSpec{})
	if err != nil {
		return nil, err
	}

	var patchedPodSpec *corev1.PodSpec
	err = json.Unmarshal(patched, &patchedPodSpec)

	return patchedPodSpec, err
}
