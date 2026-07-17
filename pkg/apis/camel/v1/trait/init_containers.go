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

// The Init Containers trait can be used to configure `init containers` or `sidecar containers`.
//
// +camel-k:trait=init-containers.
//
//nolint:godoclint
type InitContainersTrait struct {
	Trait `json:",inline" property:",squash"`

	// A list of init tasks to be executed.
	// Each task accepts the format `<name>;<container-image>;<container-command>` or key=value format
	// `name=<name>;image=<image>;command=<command>;request-cpu=<quantity>;limit-cpu=<quantity>;request-memory=<quantity>;limit-memory=<quantity>`.
	// Resource keys (request-cpu, limit-cpu, request-memory, limit-memory) are optional and accept Kubernetes resource quantities.
	InitTasks []string `json:"initTasks,omitempty" property:"init-tasks"`
	// A list of sidecar tasks to be executed.
	// Each task accepts the format `<name>;<container-image>;<container-command>` or key=value format
	// `name=<name>;image=<image>;command=<command>;request-cpu=<quantity>;limit-cpu=<quantity>;request-memory=<quantity>;limit-memory=<quantity>`.
	// Resource keys (request-cpu, limit-cpu, request-memory, limit-memory) are optional and accept Kubernetes resource quantities.
	SidecarTasks []string `json:"sideCarTasks,omitempty" property:"sidecar-tasks"`
}
