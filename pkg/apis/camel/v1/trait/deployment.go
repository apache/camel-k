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
	appsv1 "k8s.io/api/apps/v1"
)

// The Deployment trait is responsible for generating the Kubernetes deployment that will make sure
// the integration will run in the cluster.
//
// +camel-k:trait=deployment.
type DeploymentTrait struct {
	Trait `property:",squash" json:",inline"`
	// The maximum time in seconds for the deployment to make progress before it
	// is considered to be failed. It defaults to 60s.
	ProgressDeadlineSeconds *int32 `property:"progress-deadline-seconds" json:"progressDeadlineSeconds,omitempty"`
	// The deployment strategy to use to replace existing pods with new ones.
	Strategy appsv1.DeploymentStrategyType `property:"strategy" json:"strategy,omitempty"`
	// The maximum number of pods that can be unavailable during the update.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// Absolute number is calculated from percentage by rounding down.
	// This can not be 0 if MaxSurge is 0.
	// Defaults to 25%.
	RollingUpdateMaxUnavailable *int `property:"rolling-update-max-unavailable" json:"rollingUpdateMaxUnavailable,omitempty"`
	// The maximum number of pods that can be scheduled above the desired number of
	// pods.
	// Value can be an absolute number (ex: 5) or a percentage of desired pods (ex: 10%).
	// This can not be 0 if MaxUnavailable is 0.
	// Absolute number is calculated from percentage by rounding up.
	// Defaults to 25%.
	RollingUpdateMaxSurge *int `property:"rolling-update-max-surge" json:"rollingUpdateMaxSurge,omitempty"`
}
