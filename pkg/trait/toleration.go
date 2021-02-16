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

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

// This trait will enable a Toleration.
// Tolerations are applied to pods, and allow (but do not require) the pods to schedule onto nodes with matching taints.
// See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for more details.
//
// It's disabled by default.
//
// +camel-k:trait=toleration
type tolerationTrait struct {
	BaseTrait `property:",squash"`
	// Enable a toleration trait (default *false*).
	Toleration *bool `property:"toleration" json:"toleration,omitempty"`
	// The key to match the Taint
	Key string `property:"key" json:"key,omitempty"`
	// The operator (Equal | Exists)
	Operator string `property:"operator" json:"operator,omitempty"`
	// The value to match if Equal operator selected
	Value string `property:"value" json:"value,omitempty"`
	// The effect that will be set on the Pod (NoExecute | NoSchedule | PreferNoSchedule)
	Effect string `property:"effect" json:"effect,omitempty"`
	// How long that Pod stays bound to a failing or unresponsive Node
	TolerationSeconds string `property:"seconds" json:"seconds,omitempty"`
}

func newTolerationTrait() Trait {
	return &tolerationTrait{
		BaseTrait:  NewBaseTrait("toleration", 1200),
		Toleration: util.BoolP(false),
	}
}

func (t *tolerationTrait) Configure(e *Environment) (bool, error) {
	if util.IsNilOrFalse(t.Enabled) {
		return false, nil
	}

	if t.Operator == "Equal" && t.Value == "" {
		return false, fmt.Errorf("missing value for toleration equal operator")
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning), nil
}

func (t *tolerationTrait) Apply(e *Environment) (err error) {
	var deployment *appsv1.Deployment
	e.Resources.VisitDeployment(func(d *appsv1.Deployment) {
		if d.Name == e.Integration.Name {
			deployment = d
		}
	})
	if deployment != nil {
		if err := t.addToleration(e, deployment); err != nil {
			return err
		}
	}

	return nil
}

func (t *tolerationTrait) addToleration(_ *Environment, deployment *appsv1.Deployment) error {
	if t.Key == "" {
		return nil
	}

	toleration := corev1.Toleration{
		Key:      t.Key,
		Operator: corev1.TolerationOperator(t.Operator),
		Value:    t.Value,
		Effect:   corev1.TaintEffect(t.Effect),
	}

	if t.TolerationSeconds != "" {
		tolerationSeconds, err := strconv.ParseInt(t.TolerationSeconds, 10, 64)
		if err != nil {
			return err
		}
		toleration.TolerationSeconds = &tolerationSeconds
	}

	if deployment.Spec.Template.Spec.Tolerations == nil {
		deployment.Spec.Template.Spec.Tolerations = make([]corev1.Toleration, 0)
	}

	deployment.Spec.Template.Spec.Tolerations = append(deployment.Spec.Template.Spec.Tolerations, toleration)

	return nil
}
