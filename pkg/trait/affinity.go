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
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

type affinityTrait struct {
	BaseTrait `property:",squash"`

	PodAffinity           bool   `property:"pod-affinity"`
	PodAntiAffinity       bool   `property:"pod-anti-affinity"`
	NodeAffinityLabels    string `property:"node-affinity-labels"`
	PodAffinityLabels     string `property:"pod-affinity-labels"`
	PodAntiAffinityLabels string `property:"pod-anti-affinity-labels"`
}

func newAffinityTrait() *affinityTrait {
	return &affinityTrait{
		BaseTrait: newBaseTrait("affinity"),
	}
}

func (t *affinityTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		return false, nil
	}

	if t.PodAffinity && t.PodAntiAffinity {
		return false, fmt.Errorf("both pod affinity and pod anti-affinity can't be set simultaneously")
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *affinityTrait) Apply(e *Environment) (err error) {
	var deployment *appsv1.Deployment
	e.Resources.VisitDeployment(func(d *appsv1.Deployment) {
		if d.Name == e.Integration.Name {
			deployment = d
		}
	})
	if deployment != nil {
		if err := t.addNodeAffinity(e, deployment); err != nil {
			return err
		}
		if err := t.addPodAffinity(e, deployment); err != nil {
			return err
		}
		if err := t.addPodAntiAffinity(e, deployment); err != nil {
			return err
		}
	}

	return nil
}

func (t *affinityTrait) addNodeAffinity(_ *Environment, deployment *appsv1.Deployment) error {
	if t.NodeAffinityLabels == "" {
		return nil
	}

	nodeSelectorRequirements := make([]corev1.NodeSelectorRequirement, 0)
	selector, err := labels.Parse(t.NodeAffinityLabels)
	if err != nil {
		return err
	}
	requirements, _ := selector.Requirements()
	for _, r := range requirements {
		operator, err := operatorToNodeSelectorOperator(r.Operator())
		if err != nil {
			return err
		}
		nodeSelectorRequirement := corev1.NodeSelectorRequirement{
			Key:      r.Key(),
			Operator: operator,
			Values:   r.Values().List(),
		}
		nodeSelectorRequirements = append(nodeSelectorRequirements, nodeSelectorRequirement)
	}

	nodeAffinity := &corev1.NodeAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: nodeSelectorRequirements,
				},
			},
		},
	}

	if deployment.Spec.Template.Spec.Affinity == nil {
		deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{}
	}

	deployment.Spec.Template.Spec.Affinity.NodeAffinity = nodeAffinity

	return nil
}

func (t *affinityTrait) addPodAffinity(e *Environment, deployment *appsv1.Deployment) error {
	if !t.PodAffinity && t.PodAffinityLabels == "" {
		return nil
	}

	labelSelectorRequirements := make([]metav1.LabelSelectorRequirement, 0)
	if t.PodAffinityLabels != "" {
		selector, err := labels.Parse(t.PodAffinityLabels)
		if err != nil {
			return err
		}
		requirements, _ := selector.Requirements()
		for _, r := range requirements {
			operator, err := operatorToLabelSelectorOperator(r.Operator())
			if err != nil {
				return err
			}
			requirement := metav1.LabelSelectorRequirement{
				Key:      r.Key(),
				Operator: operator,
				Values:   r.Values().List(),
			}
			labelSelectorRequirements = append(labelSelectorRequirements, requirement)
		}
	}

	if t.PodAffinity {
		labelSelectorRequirements = append(labelSelectorRequirements, metav1.LabelSelectorRequirement{
			Key:      "camel.apache.org/integration",
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{
				e.Integration.Name,
			},
		})
	}

	podAffinity := &corev1.PodAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: labelSelectorRequirements,
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		},
	}

	if deployment.Spec.Template.Spec.Affinity == nil {
		deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{}
	}

	deployment.Spec.Template.Spec.Affinity.PodAffinity = podAffinity

	return nil
}

func (t *affinityTrait) addPodAntiAffinity(e *Environment, deployment *appsv1.Deployment) error {
	if !t.PodAntiAffinity && t.PodAntiAffinityLabels == "" {
		return nil
	}

	labelSelectorRequirements := make([]metav1.LabelSelectorRequirement, 0)
	if t.PodAntiAffinityLabels != "" {
		selector, err := labels.Parse(t.PodAntiAffinityLabels)
		if err != nil {
			return err
		}
		requirements, _ := selector.Requirements()
		for _, r := range requirements {
			operator, err := operatorToLabelSelectorOperator(r.Operator())
			if err != nil {
				return err
			}
			requirement := metav1.LabelSelectorRequirement{
				Key:      r.Key(),
				Operator: operator,
				Values:   r.Values().List(),
			}
			labelSelectorRequirements = append(labelSelectorRequirements, requirement)
		}
	}

	if t.PodAntiAffinity {
		labelSelectorRequirements = append(labelSelectorRequirements, metav1.LabelSelectorRequirement{
			Key:      "camel.apache.org/integration",
			Operator: metav1.LabelSelectorOpIn,
			Values: []string{
				e.Integration.Name,
			},
		})
	}

	podAntiAffinity := &corev1.PodAntiAffinity{
		RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
			{
				LabelSelector: &metav1.LabelSelector{
					MatchExpressions: labelSelectorRequirements,
				},
				TopologyKey: "kubernetes.io/hostname",
			},
		},
	}

	if deployment.Spec.Template.Spec.Affinity == nil {
		deployment.Spec.Template.Spec.Affinity = &corev1.Affinity{}
	}

	deployment.Spec.Template.Spec.Affinity.PodAntiAffinity = podAntiAffinity

	return nil
}

func operatorToNodeSelectorOperator(operator selection.Operator) (corev1.NodeSelectorOperator, error) {
	switch operator {
	case selection.In, selection.Equals, selection.DoubleEquals:
		return corev1.NodeSelectorOpIn, nil
	case selection.NotIn, selection.NotEquals:
		return corev1.NodeSelectorOpNotIn, nil
	case selection.Exists:
		return corev1.NodeSelectorOpExists, nil
	case selection.DoesNotExist:
		return corev1.NodeSelectorOpDoesNotExist, nil
	case selection.GreaterThan:
		return corev1.NodeSelectorOpGt, nil
	case selection.LessThan:
		return corev1.NodeSelectorOpLt, nil
	}
	return "", fmt.Errorf("unsupported node selector operator: %s", operator)
}

func operatorToLabelSelectorOperator(operator selection.Operator) (metav1.LabelSelectorOperator, error) {
	switch operator {
	case selection.In, selection.Equals, selection.DoubleEquals:
		return metav1.LabelSelectorOpIn, nil
	case selection.NotIn, selection.NotEquals:
		return metav1.LabelSelectorOpNotIn, nil
	case selection.Exists:
		return metav1.LabelSelectorOpExists, nil
	case selection.DoesNotExist:
		return metav1.LabelSelectorOpDoesNotExist, nil
	}
	return "", fmt.Errorf("unsupported label selector operator: %s", operator)
}
