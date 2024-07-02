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

// The builder trait is internally used to determine the best strategy to
// build and configure IntegrationKits.
//
// +camel-k:trait=builder.
type BuilderTrait struct {
	PlatformBaseTrait `property:",squash" json:",inline"`
	// Enable verbose logging on build components that support it (e.g. Kaniko build pod).
	// Deprecated no longer in use
	Verbose *bool `property:"verbose" json:"verbose,omitempty"`
	// A list of properties to be provided to the build task
	Properties []string `property:"properties" json:"properties,omitempty"`
	// The strategy to use, either `pod` or `routine` (default `routine`)
	// +kubebuilder:validation:Enum=pod;routine
	Strategy string `property:"strategy" json:"strategy,omitempty"`
	// Specify a base image. In order to have the application working properly it must be a container image which has a Java JDK
	// installed and ready to use on path (ie `/usr/bin/java`).
	BaseImage string `property:"base-image" json:"baseImage,omitempty"`
	// Use the incremental image build option, to reuse existing containers (default `true`)
	IncrementalImageBuild *bool `property:"incremental-image-build" json:"incrementalImageBuild,omitempty"`
	// The build order strategy to use, either `dependencies`, `fifo` or `sequential` (default is the platform default)
	// +kubebuilder:validation:Enum=dependencies;fifo;sequential
	OrderStrategy string `property:"order-strategy" json:"orderStrategy,omitempty"`
	// When using `pod` strategy, the minimum amount of CPU required by the pod builder.
	// Deprecated: use TasksRequestCPU instead with task name `builder`.
	RequestCPU string `property:"request-cpu" json:"requestCPU,omitempty"`
	// When using `pod` strategy, the minimum amount of memory required by the pod builder.
	// Deprecated: use TasksRequestCPU instead with task name `builder`.
	RequestMemory string `property:"request-memory" json:"requestMemory,omitempty"`
	// When using `pod` strategy, the maximum amount of CPU required by the pod builder.
	// Deprecated: use TasksRequestCPU instead with task name `builder`.
	LimitCPU string `property:"limit-cpu" json:"limitCPU,omitempty"`
	// When using `pod` strategy, the maximum amount of memory required by the pod builder.
	// Deprecated: use TasksRequestCPU instead with task name `builder`.
	LimitMemory string `property:"limit-memory" json:"limitMemory,omitempty"`
	// A list of references pointing to configmaps/secrets that contains a maven profile.
	// This configmap/secret is a resource of the IntegrationKit created, therefore it needs to be present in the namespace where the operator is going to create the IntegrationKit.
	// The content of the maven profile is expected to be a text containing a valid maven profile starting with `<profile>` and ending with `</profile>` that will be integrated as an inline profile in the POM.
	// Syntax: [configmap|secret]:name[/key], where name represents the resource name, key optionally represents the resource key to be filtered (default key value = profile.xml).
	MavenProfiles []string `property:"maven-profiles" json:"mavenProfiles,omitempty"`
	// A list of tasks to be executed (available only when using `pod` strategy) with format `<name>;<container-image>;<container-command>`.
	Tasks []string `property:"tasks" json:"tasks,omitempty"`
	// A list of tasks sorted by the order of execution in a csv format, ie, `<taskName1>,<taskName2>,...`.
	// Mind that you must include also the operator tasks (`builder`, `quarkus-native`, `package`, `jib`, `s2i`)
	// if you need to execute them. Useful only with `pod` strategy.
	TasksFilter string `property:"tasks-filter" json:"tasksFilter,omitempty"`
	// A list of request cpu configuration for the specific task with format `<task-name>:<request-cpu-conf>`.
	TasksRequestCPU []string `property:"tasks-request-cpu" json:"tasksRequestCPU,omitempty"`
	// A list of request memory configuration for the specific task with format `<task-name>:<request-memory-conf>`.
	TasksRequestMemory []string `property:"tasks-request-memory" json:"tasksRequestMemory,omitempty"`
	// A list of limit cpu configuration for the specific task with format `<task-name>:<limit-cpu-conf>`.
	TasksLimitCPU []string `property:"tasks-limit-cpu" json:"tasksLimitCPU,omitempty"`
	// A list of limit memory configuration for the specific task with format `<task-name>:<limit-memory-conf>`.
	TasksLimitMemory []string `property:"tasks-limit-memory" json:"tasksLimitMemory,omitempty"`
	// Defines a set of nodes the builder pod is eligible to be scheduled on, based on labels on the node.
	NodeSelector map[string]string `property:"node-selector" json:"nodeSelector,omitempty"`
	// When using `pod` strategy, annotation to use for the builder pod.
	Annotations map[string]string `property:"annotations" json:"annotations,omitempty"`
	// The list of manifest platforms to use to build a container image (default `linux/amd64`).
	ImagePlatforms []string `property:"platforms" json:"platforms,omitempty"`
}
