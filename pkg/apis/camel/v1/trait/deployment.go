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

// The Deployment trait is responsible for generating the Kubernetes deployment that will make sure
// the integration will run in the cluster.
//
// +camel-k:trait=deployment.
type DeploymentTrait struct {
	Trait `property:",squash" json:",inline"`
	// The maximum time in seconds for the deployment to make progress before it
	// is considered to be failed. It defaults to 60s.
	ProgressDeadlineSeconds *int32 `property:"progress-deadline-seconds" json:"progressDeadlineSeconds,omitempty"`
}
