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

	batchv1 "k8s.io/api/batch/v1"
	batchv1beta1 "k8s.io/api/batch/v1beta1"
	corev1 "k8s.io/api/core/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

type cronJobController struct {
	obj              *batchv1beta1.CronJob
	integration      *v1.Integration
	client           client.Client
	context          context.Context
	lastCompletedJob *batchv1.Job
}

var _ controller = &cronJobController{}

func (c *cronJobController) checkReadyCondition() (bool, error) {
	// Check latest job result
	if lastScheduleTime := c.obj.Status.LastScheduleTime; lastScheduleTime != nil && len(c.obj.Status.Active) == 0 {
		jobs := batchv1.JobList{}
		if err := c.client.List(c.context, &jobs,
			ctrl.InNamespace(c.integration.Namespace),
			ctrl.MatchingLabels{v1.IntegrationLabel: c.integration.Name},
		); err != nil {
			return true, err
		}
		t := lastScheduleTime.Time
		for i, job := range jobs.Items {
			if job.Status.Active == 0 && job.CreationTimestamp.Time.Before(t) {
				continue
			}
			c.lastCompletedJob = &jobs.Items[i]
			t = c.lastCompletedJob.CreationTimestamp.Time
		}
		if c.lastCompletedJob != nil {
			if failed := kubernetes.GetJobCondition(*c.lastCompletedJob, batchv1.JobFailed); failed != nil && failed.Status == corev1.ConditionTrue {
				setReadyCondition(c.integration, corev1.ConditionFalse, v1.IntegrationConditionLastJobFailedReason, fmt.Sprintf("last job %s failed: %s", c.lastCompletedJob.Name, failed.Message))
				c.integration.Status.Phase = v1.IntegrationPhaseError
				return true, nil
			}
		}
	}

	return false, nil
}

func (c *cronJobController) getPodSpec() corev1.PodSpec {
	return c.obj.Spec.JobTemplate.Spec.Template.Spec
}

func (c *cronJobController) updateReadyCondition(readyPods []corev1.Pod) bool {
	switch {
	case c.obj.Status.LastScheduleTime == nil:
		setReadyCondition(c.integration, corev1.ConditionTrue, v1.IntegrationConditionCronJobCreatedReason, "cronjob created")
		return true

	case len(c.obj.Status.Active) > 0:
		setReadyCondition(c.integration, corev1.ConditionTrue, v1.IntegrationConditionCronJobActiveReason, "cronjob active")
		return true

	case c.obj.Spec.SuccessfulJobsHistoryLimit != nil && *c.obj.Spec.SuccessfulJobsHistoryLimit == 0 && c.obj.Spec.FailedJobsHistoryLimit != nil && *c.obj.Spec.FailedJobsHistoryLimit == 0:
		setReadyCondition(c.integration, corev1.ConditionTrue, v1.IntegrationConditionCronJobCreatedReason, "no jobs history available")
		return true

	case c.lastCompletedJob != nil:
		if complete := kubernetes.GetJobCondition(*c.lastCompletedJob, batchv1.JobComplete); complete != nil && complete.Status == corev1.ConditionTrue {
			setReadyCondition(c.integration, corev1.ConditionTrue, v1.IntegrationConditionLastJobSucceededReason, fmt.Sprintf("last job %s completed successfully", c.lastCompletedJob.Name))
			return true
		}

	default:
		setReadyCondition(c.integration, corev1.ConditionUnknown, "", "")
	}

	return false
}
