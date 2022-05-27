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
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type deploymentController struct {
	obj         *appsv1.Deployment
	integration *v1.Integration
}

var _ controller = &deploymentController{}

func (c *deploymentController) checkReadyCondition() (bool, error) {
	// Check the Deployment progression
	if progressing := kubernetes.GetDeploymentCondition(*c.obj, appsv1.DeploymentProgressing); progressing != nil && progressing.Status == corev1.ConditionFalse && progressing.Reason == "ProgressDeadlineExceeded" {
		c.integration.Status.Phase = v1.IntegrationPhaseError
		setReadyConditionError(c.integration, progressing.Message)
		return true, nil
	}

	return false, nil
}

func (c *deploymentController) getPodSpec() corev1.PodSpec {
	return c.obj.Spec.Template.Spec
}

func (c *deploymentController) updateReadyCondition(readyPods []corev1.Pod) bool {
	replicas := int32(1)
	if r := c.integration.Spec.Replicas; r != nil {
		replicas = *r
	}
	// The Deployment status reports updated and ready replicas separately,
	// so that the number of ready replicas also accounts for older versions.
	readyReplicas := int32(len(readyPods))
	switch {
	case readyReplicas >= replicas:
		// The Integration is considered ready when the number of replicas
		// reported to be ready is larger than or equal to the specified number
		// of replicas. This avoids reporting a falsy readiness condition
		// when the Integration is being down-scaled.
		setReadyCondition(c.integration, corev1.ConditionTrue, v1.IntegrationConditionDeploymentReadyReason, fmt.Sprintf("%d/%d ready replicas", readyReplicas, replicas))
		return true

	case c.obj.Status.UpdatedReplicas < replicas:
		setReadyCondition(c.integration, corev1.ConditionFalse, v1.IntegrationConditionDeploymentProgressingReason, fmt.Sprintf("%d/%d updated replicas", c.obj.Status.UpdatedReplicas, replicas))

	default:
		setReadyCondition(c.integration, corev1.ConditionFalse, v1.IntegrationConditionDeploymentProgressingReason, fmt.Sprintf("%d/%d ready replicas", readyReplicas, replicas))
	}

	return false
}
