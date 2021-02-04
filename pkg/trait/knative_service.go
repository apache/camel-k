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

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

const (
	knativeServingClassAnnotation    = "autoscaling.knative.dev/class"
	knativeServingMetricAnnotation   = "autoscaling.knative.dev/metric"
	knativeServingTargetAnnotation   = "autoscaling.knative.dev/target"
	knativeServingMinScaleAnnotation = "autoscaling.knative.dev/minScale"
	knativeServingMaxScaleAnnotation = "autoscaling.knative.dev/maxScale"
)

// The Knative Service trait allows to configure options when running the integration as Knative service instead of
// a standard Kubernetes Deployment.
//
// Running integrations as Knative Services adds auto-scaling (and scaling-to-zero) features, but those features
// are only meaningful when the routes use a HTTP endpoint consumer.
//
// +camel-k:trait=knative-service
type knativeServiceTrait struct {
	BaseTrait `property:",squash"`
	// Configures the Knative autoscaling class property (e.g. to set `hpa.autoscaling.knative.dev` or `kpa.autoscaling.knative.dev` autoscaling).
	//
	// Refer to the Knative documentation for more information.
	Class string `property:"autoscaling-class" json:"class,omitempty"`
	// Configures the Knative autoscaling metric property (e.g. to set `concurrency` based or `cpu` based autoscaling).
	//
	// Refer to the Knative documentation for more information.
	Metric string `property:"autoscaling-metric" json:"autoscalingMetric,omitempty"`
	// Sets the allowed concurrency level or CPU percentage (depending on the autoscaling metric) for each Pod.
	//
	// Refer to the Knative documentation for more information.
	Target *int `property:"autoscaling-target" json:"autoscalingTarget,omitempty"`
	// The minimum number of Pods that should be running at any time for the integration. It's **zero** by default, meaning that
	// the integration is scaled down to zero when not used for a configured amount of time.
	//
	// Refer to the Knative documentation for more information.
	MinScale *int `property:"min-scale" json:"minScale,omitempty"`
	// An upper bound for the number of Pods that can be running in parallel for the integration.
	// Knative has its own cap value that depends on the installation.
	//
	// Refer to the Knative documentation for more information.
	MaxScale *int `property:"max-scale" json:"maxScale,omitempty"`
	// Automatically deploy the integration as Knative service when all conditions hold:
	//
	// * Integration is using the Knative profile
	// * All routes are either starting from a HTTP based consumer or a passive consumer (e.g. `direct` is a passive consumer)
	Auto *bool `property:"auto" json:"auto,omitempty"`
}

var _ ControllerStrategySelector = &knativeServiceTrait{}

func newKnativeServiceTrait() Trait {
	return &knativeServiceTrait{
		BaseTrait: NewBaseTrait("knative-service", 1400),
	}
}

// IsAllowedInProfile overrides default
func (t *knativeServiceTrait) IsAllowedInProfile(profile v1.TraitProfile) bool {
	return profile == v1.TraitProfileKnative
}

func (t *knativeServiceTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	if e.IntegrationInPhase(v1.IntegrationPhaseRunning) {
		condition := e.Integration.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable)
		return condition != nil && condition.Status == corev1.ConditionTrue, nil
	} else if !e.InPhase(v1.IntegrationKitPhaseReady, v1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if e.Resources.GetDeploymentForIntegration(e.Integration) != nil {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			fmt.Sprintf("different controller strategy used (%s)", string(ControllerStrategyDeployment)),
		)

		// A controller is already present for the integration
		return false, nil
	}

	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		e.Integration.Status.SetErrorCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			err,
		)

		return false, err
	}
	if strategy != ControllerStrategyKnativeService {
		e.Integration.Status.SetCondition(
			v1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1.IntegrationConditionKnativeServiceNotAvailableReason,
			fmt.Sprintf("different controller strategy used (%s)", string(strategy)),
		)

		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		// Check the right value for minScale, as not all services are allowed to scale down to 0
		if t.MinScale == nil {
			sources, err := kubernetes.ResolveIntegrationSources(t.Ctx, t.Client, e.Integration, e.Resources)
			if err != nil {
				e.Integration.Status.SetErrorCondition(
					v1.IntegrationConditionKnativeServiceAvailable,
					v1.IntegrationConditionKnativeServiceNotAvailableReason,
					err,
				)

				return false, err
			}

			meta := metadata.ExtractAll(e.CamelCatalog, sources)
			if !meta.ExposesHTTPServices || !meta.PassiveEndpoints {
				single := 1
				t.MinScale = &single
			}
		}
	}

	return true, nil
}

func (t *knativeServiceTrait) Apply(e *Environment) error {
	ksvc := t.getServiceFor(e)
	maps := e.computeConfigMaps()

	e.Resources.AddAll(maps)
	e.Resources.Add(ksvc)

	e.Integration.Status.SetCondition(
		v1.IntegrationConditionKnativeServiceAvailable,
		corev1.ConditionTrue,
		v1.IntegrationConditionKnativeServiceAvailableReason,
		fmt.Sprintf("Knative service name is %s", ksvc.Name),
	)

	if e.IntegrationInPhase(v1.IntegrationPhaseRunning) {
		replicas := e.Integration.Spec.Replicas

		isUpdateRequired := false
		minScale, ok := ksvc.Spec.Template.Annotations[knativeServingMinScaleAnnotation]
		if ok {
			min, err := strconv.Atoi(minScale)
			if err != nil {
				return err
			}
			if replicas == nil || min != int(*replicas) {
				isUpdateRequired = true
			}
		} else if replicas != nil {
			isUpdateRequired = true
		}

		maxScale, ok := ksvc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation]
		if ok {
			max, err := strconv.Atoi(maxScale)
			if err != nil {
				return err
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
					ksvc.Spec.Template.Annotations[knativeServingMinScaleAnnotation] = strconv.Itoa(*t.MinScale)
				} else {
					delete(ksvc.Spec.Template.Annotations, knativeServingMinScaleAnnotation)
				}
				if t.MaxScale != nil && *t.MaxScale > 0 {
					ksvc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation] = strconv.Itoa(*t.MaxScale)
				} else {
					delete(ksvc.Spec.Template.Annotations, knativeServingMaxScaleAnnotation)
				}
			} else {
				scale := strconv.Itoa(int(*replicas))
				ksvc.Spec.Template.Annotations[knativeServingMinScaleAnnotation] = scale
				ksvc.Spec.Template.Annotations[knativeServingMaxScaleAnnotation] = scale
			}
		}
	}

	return nil
}

func (t *knativeServiceTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	knativeServiceStrategy := ControllerStrategyKnativeService
	if t.Enabled != nil {
		if *t.Enabled {
			return &knativeServiceStrategy, nil
		}
		return nil, nil
	}

	var sources []v1.SourceSpec
	var err error
	if sources, err = kubernetes.ResolveIntegrationSources(t.Ctx, t.Client, e.Integration, e.Resources); err != nil {
		return nil, err
	}

	meta := metadata.ExtractAll(e.CamelCatalog, sources)
	if meta.ExposesHTTPServices {
		return &knativeServiceStrategy, nil
	}
	return nil, nil
}

func (t *knativeServiceTrait) ControllerStrategySelectorOrder() int {
	return 100
}

func (t *knativeServiceTrait) getServiceFor(e *Environment) *serving.Service {
	labels := map[string]string{
		v1.IntegrationLabel: e.Integration.Name,
	}

	annotations := make(map[string]string)

	// Copy annotations from the integration resource
	if e.Integration.Annotations != nil {
		for k, v := range FilterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	//
	// Set Knative Scaling behavior
	//
	if t.Class != "" {
		annotations[knativeServingClassAnnotation] = t.Class
	}
	if t.Metric != "" {
		annotations[knativeServingMetricAnnotation] = t.Metric
	}
	if t.Target != nil {
		annotations[knativeServingTargetAnnotation] = strconv.Itoa(*t.Target)
	}
	if t.MinScale != nil && *t.MinScale > 0 {
		annotations[knativeServingMinScaleAnnotation] = strconv.Itoa(*t.MinScale)
	}
	if t.MaxScale != nil && *t.MaxScale > 0 {
		annotations[knativeServingMaxScaleAnnotation] = strconv.Itoa(*t.MaxScale)
	}

	svc := serving.Service{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Service",
			APIVersion: serving.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:        e.Integration.Name,
			Namespace:   e.Integration.Namespace,
			Labels:      labels,
			Annotations: e.Integration.Annotations,
		},
		Spec: serving.ServiceSpec{
			ConfigurationSpec: serving.ConfigurationSpec{
				Template: serving.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      labels,
						Annotations: annotations,
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

	return &svc
}
