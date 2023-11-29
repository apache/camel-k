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
	"context"
	"fmt"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

type deploymentController struct {
	obj         *appsv1.Deployment
	integration *v1.Integration
}

var _ controller = &deploymentController{}

func (c *deploymentController) checkReadyCondition(ctx context.Context) (bool, error) {
	// Check the Deployment progression
	progressing := kubernetes.GetDeploymentCondition(*c.obj, appsv1.DeploymentProgressing)
	replicaFailure := kubernetes.GetDeploymentCondition(*c.obj, appsv1.DeploymentReplicaFailure)

	if replicaFailure != nil && replicaFailure.Status == corev1.ConditionTrue {
		c.integration.Status.Phase = v1.IntegrationPhaseError
		c.integration.SetReadyConditionError(replicaFailure.Message)
		return true, nil
	}

	if progressing != nil && progressing.Status == corev1.ConditionFalse && progressing.Reason == "ProgressDeadlineExceeded" {
		c.integration.Status.Phase = v1.IntegrationPhaseError
		c.integration.SetReadyConditionError(progressing.Message)
		return true, nil
	}

	return false, nil
}

func (c *deploymentController) getPodSpec() corev1.PodSpec {
	return c.obj.Spec.Template.Spec
}

func (c *deploymentController) updateReadyCondition(readyPods int) bool {
	replicas := int32(1)
	if r := c.integration.Spec.Replicas; r != nil {
		replicas = *r
	}
	// The Deployment status reports updated and ready replicas separately,
	// so that the number of ready replicas also accounts for older versions.
	readyReplicas := int32(readyPods)
	switch {
	case readyReplicas >= replicas:
		// The Integration is considered ready when the number of replicas
		// reported to be ready is larger than or equal to the specified number
		// of replicas. This avoids reporting a falsy readiness condition
		// when the Integration is being down-scaled.
		c.integration.SetReadyCondition(corev1.ConditionTrue,
			v1.IntegrationConditionDeploymentReadyReason,
			fmt.Sprintf("%d/%d ready replicas", readyReplicas, replicas))
		return true

	case c.obj.Status.UpdatedReplicas < replicas:
		c.integration.SetReadyCondition(corev1.ConditionFalse,
			v1.IntegrationConditionDeploymentProgressingReason,
			fmt.Sprintf("%d/%d updated replicas", c.obj.Status.UpdatedReplicas, replicas))

	default:
		c.integration.SetReadyCondition(corev1.ConditionFalse,
			v1.IntegrationConditionDeploymentProgressingReason,
			fmt.Sprintf("%d/%d ready replicas", readyReplicas, replicas))
	}

	return false
}

func (c *deploymentController) getSelector() metav1.LabelSelector {
	return *c.obj.Spec.Selector
}

func (c *deploymentController) isEmptySelector() bool {
	return c.obj.Spec.Selector.MatchExpressions == nil && c.obj.Spec.Selector.MatchLabels == nil
}

func (c *deploymentController) getControllerName() string {
	return fmt.Sprintf("Deployment/%s", c.obj.Name)
}
