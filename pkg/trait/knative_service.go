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
	"k8s.io/utils/ptr"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/metadata"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/knative"
)

const (
	knativeServiceTraitID               = "knative-service"
	knativeServiceTraitOrder            = 1400
	knativeServiceStrategySelectorOrder = 100

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
		BaseTrait: NewBaseTrait(knativeServiceTraitID, knativeServiceTraitOrder),
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
	if !ptr.Deref(t.Enabled, true) {
		return false, NewIntegrationCondition(
			"KnativeService",
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
		return false, nil, nil
	}

	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		return false, NewIntegrationCondition(
			"KnativeService",
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			err.Error(),
		), err
	}

	if strategy == ControllerStrategyKnativeService {
		t.Enabled = ptr.To(true)
	} else if e.IntegrationInPhase(v1.IntegrationPhaseRunning, v1.IntegrationPhaseError) {
		condition := e.Integration.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable)
		t.Enabled = ptr.To(condition != nil && condition.Status == corev1.ConditionTrue)
	}

	return ptr.Deref(t.Enabled, false), nil, nil
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
	if !ptr.Deref(t.Enabled, true) {
		// explicitly disabled by the user
		return nil, nil
	}
	// Knative serving is required
	if ok, _ := knative.IsServingInstalled(e.Client); !ok {
		if ptr.Deref(t.Enabled, false) {
			// Warn the user that he requested a feature but it cannot be fulfilled due to missing
			// API installation
			return nil, fmt.Errorf("missing Knative Service API, cannot enable Knative service trait")
		}
		// Fallback to other strategies otherwise
		return nil, nil
	}

	controllerStrategy := ControllerStrategyKnativeService
	if ptr.Deref(t.Enabled, false) {
		return &controllerStrategy, nil
	}

	enabled, err := e.ConsumeMeta(false, func(meta metadata.IntegrationMetadata) bool {
		return meta.ExposesHTTPServices || meta.PassiveEndpoints
	})
	if err != nil {
		return nil, err
	}
	if enabled {
		controllerStrategy := ControllerStrategyKnativeService
		return &controllerStrategy, nil
	}

	return nil, nil
}

func (t *knativeServiceTrait) ControllerStrategySelectorOrder() int {
	return knativeServiceStrategySelectorOrder
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
		"bindings.knative.dev/include": boolean.TrueString,
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

	if t.TimeoutSeconds != nil {
		svc.Spec.ConfigurationSpec.Template.Spec.TimeoutSeconds = t.TimeoutSeconds
	}
	replicas := e.Integration.Spec.Replicas

	isUpdateRequired := false
	minScale, ok := svc.Spec.Template.Annotations[knativeServingMinScaleAnnotation]
	if ok {
		mnScale, err := strconv.Atoi(minScale)
		if err != nil {
			return nil, err
		}
		if replicas == nil || mnScale != int(*replicas) {
			isUpdateRequired = true
		}
	} else if replicas != nil {
		isUpdateRequired = true
	}

	maxScale, ok := svc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation]
	if ok {
		mxScale, err := strconv.Atoi(maxScale)
		if err != nil {
			return nil, err
		}
		if replicas == nil || mxScale != int(*replicas) {
			isUpdateRequired = true
		}
	} else if replicas != nil {
		isUpdateRequired = true
	}

	//nolint:nestif
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
