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

// The deployer trait is responsible for deploying the resources owned by the integration, and can be used
// to explicitly select the underlying controller that will manage the integration pods.
//
// +camel-k:trait=deployer.
type DeployerTrait struct {
	Trait `property:",squash" json:",inline"`
	// Allows to explicitly select the desired deployment kind between `deployment`, `cron-job` or `knative-service` when creating the resources for running the integration.
	Kind string `property:"kind" json:"kind,omitempty"`
	// Use server-side apply to update the owned resources (default `true`).
	// Note that it automatically falls back to client-side patching, if SSA is not available, e.g., on old Kubernetes clusters.
	UseSSA *bool `property:"use-ssa" json:"useSSA,omitempty"`
}
