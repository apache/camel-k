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

package kubernetes

import (
	appsv1 "k8s.io/api/apps/v1"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	knative "knative.dev/pkg/apis"
	servingv1 "knative.dev/serving/pkg/apis/serving/v1"
)

func GetPodCondition(pod corev1.Pod, conditionType corev1.PodConditionType) *corev1.PodCondition {
	for i := range pod.Status.Conditions {
		condition := pod.Status.Conditions[i]
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func GetDeploymentCondition(deployment appsv1.Deployment,
	conditionType appsv1.DeploymentConditionType) *appsv1.DeploymentCondition {
	for i := range deployment.Status.Conditions {
		condition := deployment.Status.Conditions[i]
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func GetKnativeServiceCondition(service servingv1.Service, conditionType knative.ConditionType) *knative.Condition {
	for i := range service.Status.Conditions {
		condition := service.Status.Conditions[i]
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}

func GetJobCondition(job batchv1.Job, conditionType batchv1.JobConditionType) *batchv1.JobCondition {
	for i := range job.Status.Conditions {
		condition := job.Status.Conditions[i]
		if condition.Type == conditionType {
			return &condition
		}
	}
	return nil
}
