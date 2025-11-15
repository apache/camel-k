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

// The Master trait allows to configure the integration to automatically leverage Kubernetes resources for doing
// leader election and starting *master* routes only on certain instances.
//
// It's activated automatically when using the master endpoint in a route, e.g. `from("master:lockname:telegram:bots")...`.
//
// NOTE: this trait adds special permissions to the integration service account in order to read/write configmaps and read pods.
// It's recommended to use a different service account than "default" when running the integration.
//
// +camel-k:trait=master.
type MasterTrait struct {
	Trait `json:",inline" property:",squash"`

	// Enables automatic configuration of the trait.
	Auto *bool `json:"auto,omitempty" property:"auto"`
	// When this flag is active, the operator analyzes the source code to add dependencies required by delegate endpoints.
	// E.g. when using `master:lockname:timer`, then `camel:timer` is automatically added to the set of dependencies.
	// It's enabled by default.
	IncludeDelegateDependencies *bool `json:"includeDelegateDependencies,omitempty" property:"include-delegate-dependencies"`
	// Name of the configmap that will be used to store the lock. Defaults to "<integration-name>-lock".
	// Name of the configmap/lease resource that will be used to store the lock. Defaults to "<integration-name>-lock".
	ResourceName *string `json:"resourceName,omitempty" property:"resource-name"`
	// Type of Kubernetes resource to use for locking ("ConfigMap" or "Lease"). Defaults to "Lease".
	ResourceType *string `json:"resourceType,omitempty" property:"resource-type"`
	// Label that will be used to identify all pods contending the lock. Defaults to "camel.apache.org/integration".
	LabelKey *string `json:"labelKey,omitempty" property:"label-key"`
	// Label value that will be used to identify all pods contending the lock. Defaults to the integration name.
	LabelValue *string `json:"labelValue,omitempty" property:"label-value"`
}
