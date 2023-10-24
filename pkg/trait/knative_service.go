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
	"strconv"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

const (
	knativeServiceTraitID = "knative-service"

	// Auto-scaling annotations.
	knativeServingClassAnnotation    = "autoscaling.knative.dev/class"
	knativeServingMetricAnnotation   = "autoscaling.knative.dev/metric"
	knativeServingTargetAnnotation   = "autoscaling.knative.dev/target"
	knativeServingMinScaleAnnotation = "autoscaling.knative.dev/minScale"
	knativeServingMaxScaleAnnotation = "autoscaling.knative.dev/maxScale"
	// Rollout annotation.
	knativeServingRolloutDurationAnnotation = "serving.knative.dev/rolloutDuration"
	// visibility label.
	knativeServingVisibilityLabel = "networking.knative.dev/visibility"
)

type knativeServiceTrait struct {
	BaseTrait
	traitv1.KnativeServiceTrait `property:",squash"`
}

var _ ControllerStrategySelector = &knativeServiceTrait{}

func newKnativeServiceTrait() Trait {
	return &knativeServiceTrait{
		BaseTrait: NewBaseTrait(knativeServiceTraitID, 1400),
		KnativeServiceTrait: traitv1.KnativeServiceTrait{
			Annotations: map[string]string{},
		},
	}
}

// IsAllowedInProfile overrides default.
func (t *knativeServiceTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile.Equal(v1.TraitProfileKnative)
}

func (t *knativeServiceTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, NewIntegrationCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			"explicitly disabled",
		), nil
	}

	if !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	if e.Resources.GetDeploymentForIntegration(e.Integration) != nil {
		// A controller is already present for the integration
		return false, NewIntegrationCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			fmt.Sprintf("different controller strategy used (%s)", string(ControllerStrategyDeployment)),
		), nil
	}

	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		return false, NewIntegrationCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			err.Error(),
		), err
	}
	if strategy != ControllerStrategyKnativeService {
		return false, NewIntegrationCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			fmt.Sprintf("different controller strategy used (%s)", string(strategy)),
		), nil
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseRunning, v1.IntegrationPhaseError) {
		condition := e.Integration.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable)
		return condition != nil && condition.Status == corev1.ConditionTrue, nil, nil
	}

	return true, nil, nil
}

func (t *knativeServiceTrait) Apply(e *Environment) error {
	ksvc, err := t.getServiceFor(e)
	if err != nil {
		return err
	}
	e.Resources.Add(ksvc)

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionKnativeServiceAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionKnativeServiceAvailableReason,
		fmt.Sprintf("Knative service name is %s", ksvc.Name),
	)

	return nil
}

func (t *knativeServiceTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		// explicitly disabled
		return nil, nil
	}

	var sources []v1.SourceSpec
	var err error
	if sources, err = kubernetes.ResolveIntegrationSources(e.Ctx, t.Client, e.Integration, e.Resources); err != nil {
		return nil, err
	}

	meta, err := metadata.ExtractAll(e.CamelCatalog, sources)
	if err != nil {
		return nil, err
	}
	if meta.ExposesHTTPServices || meta.PassiveEndpoints {
		knativeServiceStrategy := ControllerStrategyKnativeService
		return &knativeServiceStrategy, nil
	}
	return nil, nil
}

func (t *knativeServiceTrait) ControllerStrategySelectorOrder() int {
	return 100
}

func (t *knativeServiceTrait) getServiceFor(e *Environment) (*serving.Service, error) {
	serviceAnnotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for k, v := range e.Integration.Annotations {
			serviceAnnotations[k] = v
		}
	}
	// Set Knative rollout
	if t.RolloutDuration != "" {
		serviceAnnotations[knativeServingRolloutDurationAnnotation] = t.RolloutDuration
	}
	if t.Annotations != nil {
		for k, v := range t.Annotations {
			serviceAnnotations[k] = v
		}
	}

	revisionAnnotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for k, v := range filterTransferableAnnotations(e.Integration.Annotations) {
			revisionAnnotations[k] = v
		}
	}
	// Set Knative auto-scaling
	if t.Class != "" {
		revisionAnnotations[knativeServingClassAnnotation] = t.Class
	}
	if t.Metric != "" {
		revisionAnnotations[knativeServingMetricAnnotation] = t.Metric
	}
	if t.Target != nil {
		revisionAnnotations[knativeServingTargetAnnotation] = strconv.Itoa(*t.Target)
	}
	if t.MinScale != nil && *t.MinScale > 0 {
		revisionAnnotations[knativeServingMinScaleAnnotation] = strconv.Itoa(*t.MinScale)
	}
	if t.MaxScale != nil && *t.MaxScale > 0 {
		revisionAnnotations[knativeServingMaxScaleAnnotation] = strconv.Itoa(*t.MaxScale)
	}

	serviceLabels := map[string]string{
		v1.IntegrationLabel: e.Integration.Name,
		// Make sure the Eventing webhook will select the source resource,
		// in order to inject the sink information.
		// This is necessary for Knative environments, that are configured
		// with SINK_BINDING_SELECTION_MODE=inclusion.
		// See:
		// - https://knative.dev/v1.3-docs/eventing/custom-event-source/sinkbinding/create-a-sinkbinding/#optional-choose-sinkbinding-namespace-selection-behavior
		// - https://github.com/knative/operator/blob/release-1.2/docs/configuration.md#specsinkbindingselectionmode
		"bindings.knative.dev/include": "true",
	}
	if t.Visibility != "" {
		serviceLabels[knativeServingVisibilityLabel] = t.Visibility
	}

	svc := serving.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: serving.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Integration.Name,
			Namespace:   e.Integration.Namespace,
			Labels:      serviceLabels,
			Annotations: serviceAnnotations,
		},
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels: map[string]string{
							v1.IntegrationLabel: e.Integration.Name,
						},
						Annotations: revisionAnnotations,
					},
					Spec: serving.RevisionSpec{
						PodSpec: corev1.PodSpec{
							ServiceAccountName: e.Integration.Spec.ServiceAccountName,
						},
					},
				},
			},
		},
	}

	replicas := e.Integration.Spec.Replicas

	isUpdateRequired := false
	minScale, ok := svc.Spec.Template.Annotations[knativeServingMinScaleAnnotation]
	if ok {
		min, err := strconv.Atoi(minScale)
		if err != nil {
			return nil, err
		}
		if replicas == nil || min != int(*replicas) {
			isUpdateRequired = true
		}
	} else if replicas != nil {
		isUpdateRequired = true
	}

	maxScale, ok := svc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation]
	if ok {
		max, err := strconv.Atoi(maxScale)
		if err != nil {
			return nil, err
		}
		if replicas == nil || max != int(*replicas) {
			isUpdateRequired = true
		}
	} else if replicas != nil {
		isUpdateRequired = true
	}

	if isUpdateRequired {
		if replicas == nil {
			if t.MinScale != nil && *t.MinScale > 0 {
				svc.Spec.Template.Annotations[knativeServingMinScaleAnnotation] = strconv.Itoa(*t.MinScale)
			} else {
				delete(svc.Spec.Template.Annotations, knativeServingMinScaleAnnotation)
			}
			if t.MaxScale != nil && *t.MaxScale > 0 {
				svc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation] = strconv.Itoa(*t.MaxScale)
			} else {
				delete(svc.Spec.Template.Annotations, knativeServingMaxScaleAnnotation)
			}
		} else {
			scale := strconv.Itoa(int(*replicas))
			svc.Spec.Template.Annotations[knativeServingMinScaleAnnotation] = scale
			svc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation] = scale
		}
	}

	return &svc, nil
}
