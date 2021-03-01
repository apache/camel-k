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
	"regexp"
	"strconv"

	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util"
)

// This trait sets Tolerations over Integration pods. Tolerations allow (but do not require) the pods to schedule onto nodes with matching taints.
// See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for more details.
//
// The toleration should be expressed in a similar manner of taints *_Key[=Value]:Effect[:Seconds]_* where values in square brackets are optional. Examples:
//
// node-role.kubernetes.io/master:NoSchedule
// node.kubernetes.io/network-unavailable:NoExecute:3000
// disktype=ssd:PreferNoSchedule
//
// It's disabled by default.
//
// +camel-k:trait=toleration
type tolerationTrait struct {
	BaseTrait `property:",squash"`
	// The taint to tolerate in the form Key[=Value]:Effect[:Seconds]
	Taints []string `property:"taints" json:"taints,omitempty"`
}

var validTaintRegexp = regexp.MustCompile(`^([\w\/_\-\.]+)(=)?([\w_\-\.]+)?:(NoSchedule|NoExecute|PreferNoSchedule):?(\d*)?$`)

func newTolerationTrait() Trait {
	return &tolerationTrait{
		BaseTrait: NewBaseTrait("toleration", 1200),
	}
}

func (t *tolerationTrait) Configure(e *Environment) (bool, error) {
	if util.IsNilOrFalse(t.Enabled) {
		return false, nil
	}

	if len(t.Taints) == 0 {
		return false, fmt.Errorf("no taint was provided")
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseDeploying, v1.IntegrationPhaseRunning), nil
}

func (t *tolerationTrait) Apply(e *Environment) (err error) {
	tolerations, err := t.getTolerations()
	if err != nil {
		return err
	}
	podSpec := e.GetIntegrationPodSpec()

	// Add the toleration
	if podSpec != nil {
		if podSpec.Tolerations == nil {
			podSpec.Tolerations = make([]corev1.Toleration, 0)
		}
		podSpec.Tolerations = append(podSpec.Tolerations, tolerations...)
	}

	return nil
}

func (t *tolerationTrait) getTolerations() ([]corev1.Toleration, error) {
	tolerations := make([]corev1.Toleration, 0)
	for _, t := range t.Taints {
		if !validTaintRegexp.MatchString(t) {
			return nil, fmt.Errorf("could not match taint %v", t)
		}
		toleration := corev1.Toleration{}
		// Parse the regexp groups
		groups := validTaintRegexp.FindStringSubmatch(t)
		toleration.Key = groups[1]
		if groups[2] != "" {
			toleration.Operator = corev1.TolerationOpEqual
		} else {
			toleration.Operator = corev1.TolerationOpExists
		}
		if groups[3] != "" {
			toleration.Value = groups[3]
		}
		toleration.Effect = corev1.TaintEffect(groups[4])

		if groups[5] != "" {
			tolerationSeconds, err := strconv.ParseInt(groups[5], 10, 64)
			if err != nil {
				return nil, err
			}
			toleration.TolerationSeconds = &tolerationSeconds
		}
		tolerations = append(tolerations, toleration)
	}

	return tolerations, nil
}
