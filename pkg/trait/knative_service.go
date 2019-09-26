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
	"context"
	"strconv"

	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
	serving "knative.dev/serving/pkg/apis/serving/v1beta1"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
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

type knativeServiceTrait struct {
	BaseTrait `property:",squash"`
	Class     string `property:"autoscaling-class"`
	Metric    string `property:"autoscaling-metric"`
	Target    *int   `property:"autoscaling-target"`
	MinScale  *int   `property:"min-scale"`
	MaxScale  *int   `property:"max-scale"`
	Auto      *bool  `property:"auto"`
	deployer  deployerTrait
}

func newKnativeServiceTrait() *knativeServiceTrait {
	return &knativeServiceTrait{
		BaseTrait: newBaseTrait("knative-service"),
	}
}

func (t *knativeServiceTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		e.Integration.Status.SetCondition(
			v1alpha1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1alpha1.IntegrationConditionKnativeServiceNotAvailableReason,
			"explicitly disabled",
		)

		return false, nil
	}

	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseRunning) {
		condition := e.Integration.Status.GetCondition(v1alpha1.IntegrationConditionKnativeServiceAvailable)
		return condition != nil && condition.Status == corev1.ConditionTrue, nil
	} else if !e.InPhase(v1alpha1.IntegrationKitPhaseReady, v1alpha1.IntegrationPhaseDeploying) {
		return false, nil
	}

	if e.Resources.GetDeploymentForIntegration(e.Integration) != nil {
		e.Integration.Status.SetCondition(
			v1alpha1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1alpha1.IntegrationConditionKnativeServiceNotAvailableReason,
			"controller strategy: "+ControllerStrategyDeployment,
		)

		// A controller is already present for the integration
		return false, nil
	}

	strategy, err := e.DetermineControllerStrategy(t.ctx, t.client)
	if err != nil {
		e.Integration.Status.SetErrorCondition(
			v1alpha1.IntegrationConditionKnativeServiceAvailable,
			v1alpha1.IntegrationConditionKnativeServiceNotAvailableReason,
			err,
		)

		return false, err
	}
	if strategy != ControllerStrategyKnativeService {
		e.Integration.Status.SetCondition(
			v1alpha1.IntegrationConditionKnativeServiceAvailable,
			corev1.ConditionFalse,
			v1alpha1.IntegrationConditionKnativeServiceNotAvailableReason,
			"controller strategy: "+string(strategy),
		)

		return false, nil
	}

	if t.Auto == nil || *t.Auto {
		// Check the right value for minScale, as not all services are allowed to scale down to 0
		if t.MinScale == nil {
			sources, err := kubernetes.ResolveIntegrationSources(t.ctx, t.client, e.Integration, e.Resources)
			if err != nil {
				e.Integration.Status.SetErrorCondition(
					v1alpha1.IntegrationConditionKnativeServiceAvailable,
					v1alpha1.IntegrationConditionKnativeServiceNotAvailableReason,
					err,
				)

				return false, err
			}

			meta := metadata.ExtractAll(e.CamelCatalog, sources)
			if !meta.RequiresHTTPService || !meta.PassiveEndpoints {
				single := 1
				t.MinScale = &single
			}
		}
	}

	dt := e.Catalog.GetTrait("deployer")
	if dt != nil {
		t.deployer = *dt.(*deployerTrait)
	}

	return true, nil
}

func (t *knativeServiceTrait) Apply(e *Environment) error {
	if e.IntegrationInPhase(v1alpha1.IntegrationPhaseRunning) {
		// Do not reconcile the Knative if no replicas is set
		if e.Integration.Spec.Replicas == nil {
			return nil
		}
		// Otherwise set the Knative scale annotations
		replicas := int(*e.Integration.Spec.Replicas)

		service := &serving.Service{}
		err := t.client.Get(context.TODO(), client.ObjectKey{Namespace: e.Integration.Namespace, Name: e.Integration.Name}, service)
		if err != nil {
			return err
		}

		isUpdateRequired := false
		minScale, ok := service.Spec.Template.Annotations[knativeServingMinScaleAnnotation]
		if ok {
			min, err := strconv.Atoi(minScale)
			if err != nil {
				return err
			}
			if min != replicas {
				isUpdateRequired = true
			}
		} else {
			isUpdateRequired = true
		}

		maxScale, ok := service.Spec.Template.Annotations[knativeServingMaxScaleAnnotation]
		if ok {
			max, err := strconv.Atoi(maxScale)
			if err != nil {
				return err
			}
			if max != replicas {
				isUpdateRequired = true
			}
		} else {
			isUpdateRequired = true
		}

		if isUpdateRequired {
			scale := strconv.Itoa(replicas)
			service.Spec.Template.Annotations[knativeServingMinScaleAnnotation] = scale
			service.Spec.Template.Annotations[knativeServingMaxScaleAnnotation] = scale
			err := t.client.Update(context.TODO(), service)
			if err != nil {
				return err
			}
		}

		return nil
	}

	ksvc := t.getServiceFor(e)
	maps := e.ComputeConfigMaps()

	e.Resources.AddAll(maps)
	e.Resources.Add(ksvc)

	e.Integration.Status.SetCondition(
		v1alpha1.IntegrationConditionKnativeServiceAvailable,
		corev1.ConditionTrue,
		v1alpha1.IntegrationConditionKnativeServiceAvailableReason,
		ksvc.Name,
	)

	return nil
}

func (t *knativeServiceTrait) getServiceFor(e *Environment) *serving.Service {
	labels := map[string]string{
		"camel.apache.org/integration": e.Integration.Name,
	}

	annotations := make(map[string]string)

	// Copy annotations from the integration resource
	if e.Integration.Annotations != nil {
		for k, v := range FilterTransferableAnnotations(e.Integration.Annotations) {
			annotations[k] = v
		}
	}

	// Resolve registry host names when used
	annotations["alpha.image.policy.openshift.io/resolve-names"] = "*"

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
		Spec: servingv1.ServiceSpec{
			ConfigurationSpec: servingv1.ConfigurationSpec{
				Template: servingv1.RevisionTemplateSpec{
					ObjectMeta: metav1.ObjectMeta{
						Labels:      labels,
						Annotations: annotations,
					},
					Spec: servingv1.RevisionSpec{
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
