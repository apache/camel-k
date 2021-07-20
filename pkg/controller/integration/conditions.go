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
	"strconv"

	appsv1 "k8s.io/api/apps/v1"
	"k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
)

func setReadyCondition(ctx context.Context, c client.Client, it *v1.Integration) (err error) {
	if isConditionTrue(it, v1.IntegrationConditionDeploymentAvailable) {
		set := false
		if set, err = setReadyConditionFromDeployment(ctx, c, it); err != nil {
			return
		} else if !set {
			// TODO: we may want to rely on the Deployment status, instead of on that of the latest ReplicaSet
			err = setReadyConditionFromReplicaSet(ctx, c, it)
		}
	} else if isConditionTrue(it, v1.IntegrationConditionKnativeServiceAvailable) {
		// TODO: reconciliation to Error phase?
		err = setReadyConditionFromReplicaSet(ctx, c, it)
	} else if isConditionTrue(it, v1.IntegrationConditionCronJobAvailable) {
		// TODO: reconciliation to Error phase?
		err = setReadyConditionFromCronJob(ctx, c, it)
	} else {
		it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionUnknown, "", "")
	}
	return
}

func setReadyConditionFromDeployment(ctx context.Context, c client.Client, it *v1.Integration) (bool, error) {
	deployment := &appsv1.Deployment{}
	if err := c.Get(ctx, ctrl.ObjectKeyFromObject(it), deployment); err != nil {
		if errors.IsNotFound(err) {
			setReadyConditionError(it, err.Error())
			return true, nil
		} else {
			return false, err
		}
	}
	progressing := getDeploymentCondition(deployment, appsv1.DeploymentProgressing)
	if progressing != nil && progressing.Status == corev1.ConditionFalse && progressing.Reason == "ProgressDeadlineExceeded" {
		// Report the Error reason, in case the Deployment exceeded its progress deadline
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionFalse,
			v1.IntegrationConditionErrorReason,
			fmt.Sprintf("deployment %q exceeded its progress deadline", deployment))
		return true, nil
	}

	return false, nil
}

func setReadyConditionFromReplicaSet(ctx context.Context, c client.Client, it *v1.Integration) error {
	list := appsv1.ReplicaSetList{}
	err := c.List(ctx, &list, ctrl.MatchingLabels{v1.IntegrationLabel: it.Name}, ctrl.InNamespace(it.Namespace))
	if err != nil {
		return err
	}

	if len(list.Items) == 0 {
		setReadyConditionError(it, "ReplicaSet not found")
		return nil
	}

	var rs *appsv1.ReplicaSet
	for _, r := range list.Items {
		r := r
		if r.Labels["camel.apache.org/generation"] == strconv.FormatInt(it.Generation, 10) {
			rs = &r
		}
	}
	if rs == nil {
		rs = &list.Items[0]
	}
	var replicas int32 = 1
	if rs.Spec.Replicas != nil {
		replicas = *rs.Spec.Replicas
	}
	// The Integration is considered ready when the number of replicas
	// reported to be ready is larger or equal to the specified number
	// of replicas. This avoid reporting a falsy readiness condition
	// when the Integration is being down-scaled.
	if replicas <= rs.Status.ReadyReplicas {
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionTrue,
			v1.IntegrationConditionReplicaSetReadyReason,
			"",
		)
	} else {
		it.Status.SetCondition(
			v1.IntegrationConditionReady,
			corev1.ConditionFalse,
			v1.IntegrationConditionReplicaSetNotReadyReason,
			"",
		)
	}
	return nil
}

func setReadyConditionFromCronJob(ctx context.Context, c client.Client, it *v1.Integration) error {
	cronJob := v1beta1.CronJob{}
	if err := c.Get(ctx, ctrl.ObjectKeyFromObject(it), &cronJob); err != nil {
		if errors.IsNotFound(err) {
			setReadyConditionError(it, err.Error())
			return nil
		} else {
			return err
		}
	}

	// CronJob status is not tracked by Kubernetes
	it.Status.SetCondition(
		v1.IntegrationConditionReady,
		corev1.ConditionTrue,
		v1.IntegrationConditionCronJobCreatedReason,
		"",
	)

	return nil
}

func isConditionTrue(it *v1.Integration, conditionType v1.IntegrationConditionType) bool {
	condition := it.Status.GetCondition(conditionType)
	if condition == nil {
		return false
	}
	return condition.Status == corev1.ConditionTrue
}

func setReadyConditionError(it *v1.Integration, err string) {
	it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionUnknown,
		v1.IntegrationConditionErrorReason, err)
}

func getDeploymentCondition(deployment *appsv1.Deployment, conditionType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range deployment.Status.Conditions {
		c := deployment.Status.Conditions[i]
		if c.Type == conditionType {
			return &c
		}
	}
	return nil
}
