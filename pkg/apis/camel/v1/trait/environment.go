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

// The environment trait is used internally to inject standard environment variables in the integration container,
// such as `NAMESPACE`, `POD_NAME` and others.
//
// +camel-k:trait=environment.
type EnvironmentTrait struct {
	PlatformBaseTrait `json:",inline" property:",squash"`

	// Enables injection of `NAMESPACE` and `POD_NAME` environment variables (default `true`)
	ContainerMeta *bool `json:"containerMeta,omitempty" property:"container-meta"`
	// Propagates the `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY` environment variables (default `true`)
	HTTPProxy *bool `json:"httpProxy,omitempty" property:"http-proxy"`
	// A list of environment variables to be added to the integration container.
	// The syntax is either VAR=VALUE or VAR=[configmap|secret]:name/key, where name represents the resource name,
	// and key represents the resource key to be mapped as and environment variable.
	// These take precedence over any previously defined environment variables.
	Vars []string `json:"vars,omitempty" property:"vars"`
}
