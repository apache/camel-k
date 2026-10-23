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
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

type cronJobController struct {
	obj *batchv1.CronJob
}

func (c *cronJobController) setReadyCondition(integration *v1.Integration) {
	cronJob := c.obj

	// No scheduled job yet: leave the Ready condition unchanged.
	if cronJob.Status.LastScheduleTime == nil {
		return
	}

	// A job is actively running: leave the Ready condition unchanged.
	if len(cronJob.Status.Active) > 0 {
		return
	}

	// The last scheduled job has not succeeded.
	if cronJob.Status.LastSuccessfulTime == nil ||
		cronJob.Status.LastSuccessfulTime.Before(cronJob.Status.LastScheduleTime) {
		integration.Status.Phase = v1.IntegrationPhaseError
		integration.SetReadyCondition(
			corev1.ConditionFalse,
			v1.IntegrationConditionLastJobFailedReason,
			"The last scheduled job has not succeeded",
		)

		return
	}

	// The last scheduled job succeeded.
	integration.SetReadyCondition(
		corev1.ConditionTrue,
		v1.IntegrationConditionLastJobSucceededReason,
		"The last scheduled job succeeded",
	)
}
