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
	"strings"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/selection"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// Allows constraining which nodes the integration pod(s) are eligible to be scheduled on, based on labels on the node,
// or with inter-pod affinity and anti-affinity, based on labels on pods that are already running on the nodes.
//
// It's disabled by default.
//
// +camel-k:trait=affinity
type affinityTrait struct {
	BaseTrait `property:",squash"`
	// Always co-locates multiple replicas of the integration in the same node (default *false*).
	PodAffinity *bool `property:"pod-affinity" json:"podAffinity,omitempty"`
	// Never co-locates multiple replicas of the integration in the same node (default *false*).
	PodAntiAffinity *bool `property:"pod-anti-affinity" json:"podAntiAffinity,omitempty"`
	// Defines a set of nodes the integration pod(s) are eligible to be scheduled on, based on labels on the node.
	NodeAffinityLabels []string `property:"node-affinity-labels" json:"nodeAffinityLabels,omitempty"`
	// Defines a set of pods (namely those matching the label selector, relative to the given namespace) that the
	// integration pod(s) should be co-located with.
	PodAffinityLabels []string `property:"pod-affinity-labels" json:"podAffinityLabels,omitempty"`
	// Defines a set of pods (namely those matching the label selector, relative to the given namespace) that the
	// integration pod(s) should not be co-located with.
	PodAntiAffinityLabels []string `property:"pod-anti-affinity-labels" json:"podAntiAffinityLabels,omitempty"`
}

func newAffinityTrait() Trait {
	return &affinityTrait{
		BaseTrait:       NewBaseTrait("affinity", 1300),
		PodAffinity:     &[]bool{false}[0],
		PodAntiAffinity: &[]bool{false}[0],
	}
}

func (t *affinityTrait) Configure(e *Environment) (bool, error) {
	if isNilOrFalse(t.Enabled) {
		return false, nil
	}

	if isTrue(t.PodAffinity) && isTrue(t.PodAntiAffinity) {
		return false, fmt.Errorf("both pod affinity and pod anti-affinity can't be set simultaneously")
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning), nil
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
	if len(t.NodeAffinityLabels) == 0 {
		return nil
	}

	nodeSelectorRequirements := make([]corev1.NodeSelectorRequirement, 0)
	selector, err := labels.Parse(strings.Join(t.NodeAffinityLabels, ","))
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
	if isNilOrFalse(t.PodAffinity) && len(t.PodAffinityLabels) == 0 {
		return nil
	}

	labelSelectorRequirements := make([]metav1.LabelSelectorRequirement, 0)
	if len(t.PodAffinityLabels) > 0 {
		selector, err := labels.Parse(strings.Join(t.PodAffinityLabels, ","))
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

	if isTrue(t.PodAffinity) {
		labelSelectorRequirements = append(labelSelectorRequirements, metav1.LabelSelectorRequirement{
			Key:      v1.IntegrationLabel,
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
	if isNilOrFalse(t.PodAntiAffinity) && len(t.PodAntiAffinityLabels) == 0 {
		return nil
	}

	labelSelectorRequirements := make([]metav1.LabelSelectorRequirement, 0)
	if len(t.PodAntiAffinityLabels) > 0 {
		selector, err := labels.Parse(strings.Join(t.PodAntiAffinityLabels, ","))
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

	if isTrue(t.PodAntiAffinity) {
		labelSelectorRequirements = append(labelSelectorRequirements, metav1.LabelSelectorRequirement{
			Key:      v1.IntegrationLabel,
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
