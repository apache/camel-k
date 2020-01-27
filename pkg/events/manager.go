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

package events

import (
	"fmt"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/client-go/tools/record"
)

const (
	// ReasonIntegrationPhaseUpdated --
	ReasonIntegrationPhaseUpdated = "IntegrationPhaseUpdated"
	// ReasonIntegrationConditionChanged --
	ReasonIntegrationConditionChanged = "IntegrationConditionChanged"
	// ReasonIntegrationError
	ReasonIntegrationError = "IntegrationError"
)

// NotifyIntegrationError automatically generates error events when the integration reconcile cycle phase has an error
func NotifyIntegrationError(recorder record.EventRecorder, old, new *v1.Integration, err error) {
	it := old
	if new != nil {
		it = new
	}
	if it == nil {
		return
	}
	recorder.Eventf(it, corev1.EventTypeWarning, ReasonIntegrationError, "Cannot reconcile integration %s: %v", it.Name, err)
}

// NotifyIntegrationUpdated automatically generates events when the integration changes
func NotifyIntegrationUpdated(recorder record.EventRecorder, old, new *v1.Integration) {
	if new == nil {
		return
	}

	// Update information about phase changes
	if old == nil || old.Status.Phase != new.Status.Phase {
		phase := new.Status.Phase
		if phase == v1.IntegrationPhaseNone {
			phase = "[none]"
		}
		recorder.Eventf(new, corev1.EventTypeNormal, ReasonIntegrationPhaseUpdated, "Integration %s in phase %s", new.Name, phase)
	}

	// Update information about changes in conditions
	if new.Status.Phase != v1.IntegrationPhaseNone {
		for _, cond := range getChangedConditions(old, new) {
			head := ""
			if cond.Status == corev1.ConditionFalse {
				head = "No "
			}
			tail := ""
			if cond.Message != "" {
				tail = fmt.Sprintf(": %s", cond.Message)
			}
			recorder.Eventf(new, corev1.EventTypeNormal, ReasonIntegrationConditionChanged, "%s%s for integration %s%s", head, cond.Type, new.Name, tail)
		}
	}

}

func getChangedConditions(old, new *v1.Integration) (res []v1.IntegrationCondition) {
	if old == nil {
		old = &v1.Integration{}
	}
	if new == nil {
		new = &v1.Integration{}
	}
	for _, newCond := range new.Status.Conditions {
		oldCond := old.Status.GetCondition(newCond.Type)
		if oldCond == nil || oldCond.Status != newCond.Status || oldCond.Message != newCond.Message {
			res = append(res, newCond)
		}
	}
	return res
}
