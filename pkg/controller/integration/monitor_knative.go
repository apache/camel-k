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

package integration

import (
	corev1 "k8s.io/api/core/v1"

	servingv1 "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type knativeServiceController struct {
	obj         *servingv1.Service
	integration *v1.Integration
}

var _ controller = &knativeServiceController{}

func (c *knativeServiceController) checkReadyCondition() (bool, error) {
	// Check the KnativeService conditions
	if ready := kubernetes.GetKnativeServiceCondition(*c.obj, servingv1.ServiceConditionReady); ready.IsFalse() && ready.GetReason() == "RevisionFailed" {
		c.integration.Status.Phase = v1.IntegrationPhaseError
		setReadyConditionError(c.integration, ready.Message)
		return true, nil
	}

	return false, nil
}

func (c *knativeServiceController) getPodSpec() corev1.PodSpec {
	return c.obj.Spec.Template.Spec.PodSpec
}

func (c *knativeServiceController) updateReadyCondition(readyPods []corev1.Pod) bool {
	ready := kubernetes.GetKnativeServiceCondition(*c.obj, servingv1.ServiceConditionReady)
	if ready.IsTrue() {
		setReadyCondition(c.integration, corev1.ConditionTrue, v1.IntegrationConditionKnativeServiceReadyReason, "")
		return true
	}
	setReadyCondition(c.integration, corev1.ConditionFalse, ready.GetReason(), ready.GetMessage())

	return false
}
