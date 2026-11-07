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
// +camel-k:internal.
//
//nolint:godoclint
type DeployerTrait struct {
	PlatformBaseTrait `json:",inline" property:",squash"`

	// Deprecated: no longer in use.
	DeprecatedKind string `json:"kind,omitempty" property:"kind"`
	// Deprecated: no longer in use.
	DeprecatedUseSSA *bool `json:"useSSA,omitempty" property:"use-ssa"`
}
