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

	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// The PDB trait allows to configure the PodDisruptionBudget resource for the Integration pods.
//
// +camel-k:trait=pdb
type pdbTrait struct {
	BaseTrait `property:",squash"`
	// The number of pods for the Integration that must still be available after an eviction.
	// It can be either an absolute number or a percentage.
	// Only one of `min-available` and `max-unavailable` can be specified.
	MinAvailable string `property:"min-available" json:"minAvailable,omitempty"`
	// The number of pods for the Integration that can be unavailable after an eviction.
	// It can be either an absolute number or a percentage (default `1` if `min-available` is also not set).
	// Only one of `max-unavailable` and `min-available` can be specified.
	MaxUnavailable string `property:"max-unavailable" json:"maxUnavailable,omitempty"`
}

func newPdbTrait() Trait {
	return &pdbTrait{
		BaseTrait: NewBaseTrait("pdb", 900),
	}
}

func (t *pdbTrait) Configure(e *Environment) (bool, error) {
	if IsNilOrFalse(t.Enabled) {
		return false, nil
	}

	strategy, err := e.DetermineControllerStrategy()
	if err != nil {
		return false, fmt.Errorf("unable to determine the controller stratedy")
	}

	if strategy == ControllerStrategyCronJob {
		return false, fmt.Errorf("poddisruptionbudget isn't supported with cron-job controller strategy")
	}

	if t.MaxUnavailable != "" && t.MinAvailable != "" {
		return false, fmt.Errorf("both minAvailable and maxUnavailable can't be set simultaneously")
	}

	return e.IntegrationInPhase(
		v1.IntegrationPhaseDeploying,
		v1.IntegrationPhaseRunning,
	), nil
}

func (t *pdbTrait) Apply(e *Environment) error {
	if t.MaxUnavailable == "" && t.MinAvailable == "" {
		t.MaxUnavailable = "1"
	}

	pdb := t.podDisruptionBudgetFor(e.Integration)
	e.Resources.Add(pdb)

	return nil
}

func (t *pdbTrait) podDisruptionBudgetFor(integration *v1.Integration) *v1beta1.PodDisruptionBudget {
	pdb := &v1beta1.PodDisruptionBudget{
		TypeMeta: metav1.TypeMeta{
			Kind:       "PodDisruptionBudget",
			APIVersion: v1beta1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      integration.Name,
			Namespace: integration.Namespace,
			Labels:    integration.Labels,
		},
		Spec: v1beta1.PodDisruptionBudgetSpec{
			Selector: &metav1.LabelSelector{
				MatchLabels: map[string]string{
					v1.IntegrationLabel: integration.Name,
				},
			},
		},
	}

	if t.MaxUnavailable != "" {
		max := intstr.Parse(t.MaxUnavailable)
		pdb.Spec.MaxUnavailable = &max
	} else {
		min := intstr.Parse(t.MinAvailable)
		pdb.Spec.MinAvailable = &min
	}

	return pdb
}
