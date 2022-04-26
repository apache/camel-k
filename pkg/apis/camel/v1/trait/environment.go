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
	Trait `property:",squash" json:",inline"`
	// Enables injection of `NAMESPACE` and `POD_NAME` environment variables (default `true`)
	ContainerMeta *bool `property:"container-meta" json:"containerMeta,omitempty"`
	// Propagates the `HTTP_PROXY`, `HTTPS_PROXY` and `NO_PROXY` environment variables (default `true`)
	HTTPProxy *bool `property:"http-proxy" json:"httpProxy,omitempty"`
	// A list of environment variables to be added to the integration container.
	// The syntax is KEY=VALUE, e.g., `MY_VAR="my value"`.
	// These take precedence over the previously defined environment variables.
	Vars []string `property:"vars" json:"vars,omitempty"`
}
