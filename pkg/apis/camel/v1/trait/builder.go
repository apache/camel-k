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

// BuilderTrait is internally used to determine the best strategy to
// build and configure IntegrationKits.
//
// +camel-k:trait=builder.
type BuilderTrait struct {
	Trait `property:",squash" json:",inline"`
	// Enable verbose logging on build components that support it (e.g. Kaniko build pod).
	Verbose *bool `property:"verbose" json:"verbose,omitempty"`
	// A list of properties to be provided to the build task
	Properties []string `property:"properties" json:"properties,omitempty"`
	// The strategy to use, either `pod` or `routine` (default routine)
	Strategy string `property:"strategy" json:"strategy,omitempty"`
	// When using `pod` strategy, the minimum amount of CPU required by the pod builder.
	RequestCPU string `property:"request-cpu" json:"requestCPU,omitempty"`
	// When using `pod` strategy, the minimum amount of memory required by the pod builder.
	RequestMemory string `property:"request-memory" json:"requestMemory,omitempty"`
	// When using `pod` strategy, the maximum amount of CPU required by the pod builder.
	LimitCPU string `property:"limit-cpu" json:"limitCPU,omitempty"`
	// When using `pod` strategy, the maximum amount of memory required by the pod builder.
	LimitMemory string `property:"limit-memory" json:"limitMemory,omitempty"`
	// A list of tasks to be executed (available only when using `pod` strategy) with format <name>;<container-image>;<container-command>
	Tasks []string `property:"tasks" json:"tasks,omitempty"`
}

// A Task is a generic operation run on the project
type Task struct {
	// The name of the task to execute
	Name string `property:"name" json:"name,omitempty"`
	// The container image to use
	Image string `property:"image" json:"image,omitempty"`
	// The command to execute
	Command string `property:"command" json:"command,omitempty"`
}
