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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"k8s.io/api/policy/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

// The Pdb trait allows to configure the PodDisruptionBudget resource.
//
// +camel-k:trait=pdb
type pdbTrait struct {
	BaseTrait      `property:",squash"`
	MaxUnavailable string `property:"max-unavailable" json:"maxUnavailable,omitempty"`
	MinAvailable   string `property:"min-available" json:"minAvailable,omitempty"`
}

func newPdbTrait() Trait {
	return &pdbTrait{
		BaseTrait: NewBaseTrait("pdb", 900),
	}
}

func (t *pdbTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
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
	if pdb, err := t.generatePodDisruptionBudget(e); err == nil {
		e.Resources.Add(pdb)
	} else {
		return err
	}
	return nil
}

func (t *pdbTrait) generatePodDisruptionBudget(e *Environment) (*v1beta1.PodDisruptionBudget, error) {
	if t.MaxUnavailable == "" && t.MinAvailable == "" {
		t.MaxUnavailable = "1"
	}

	integration := e.Integration
	spec := v1beta1.PodDisruptionBudgetSpec{
		Selector: &metav1.LabelSelector{
			MatchLabels: map[string]string{
				v1.IntegrationLabel: integration.Name,
			},
		},
	}

	var min, max intstr.IntOrString

	if t.MaxUnavailable != "" {
		max = intstr.Parse(t.MaxUnavailable)
		spec.MaxUnavailable = &max

	} else {
		min = intstr.Parse(t.MinAvailable)
		spec.MinAvailable = &min
	}

	return &v1beta1.PodDisruptionBudget{
		ObjectMeta: metav1.ObjectMeta{
			Name:      integration.Name,
			Namespace: integration.Namespace,
			Labels:    integration.Labels,
		},
		Spec: spec,
	}, nil
}
