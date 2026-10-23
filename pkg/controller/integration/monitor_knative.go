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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	servingv1 "github.com/apache/camel-k/v2/pkg/apis/duck/knative/serving/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

type knativeServiceController struct {
	obj *servingv1.Service
}

func (c *knativeServiceController) setReadyCondition(integration *v1.Integration) {
	// Check the KnativeService conditions
	if ready := kubernetes.GetKnativeServiceCondition(*c.obj, servingv1.ServiceConditionReady); ready != nil &&
		ready.Status == metav1.ConditionFalse && ready.Reason == "RevisionFailed" {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyConditionError(ready.Message)

		return
	}

	integration.SetReadyCondition(corev1.ConditionTrue,
		v1.IntegrationConditionKnativeServiceReadyReason, "Ready")
}
