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

// The Mount trait can be used to configure volumes mounted on the Integration Pods.
//
// +camel-k:trait=mount
// nolint: tagliatelle
type MountTrait struct {
	Trait `property:",squash" json:",inline"`
	// A list of configuration pointing to configmap/secret.
	// The configuration are expected to be UTF-8 resources as they are processed by runtime Camel Context and tried to be parsed as property files.
	// They are also made available on the classpath in order to ease their usage directly from the Route.
	// Syntax: [configmap|secret]:name[/key], where name represents the resource name and key optionally represents the resource key to be filtered
	Configs []string `property:"configs" json:"configs,omitempty"`
	// A list of resources (text or binary content) pointing to configmap/secret.
	// The resources are expected to be any resource type (text or binary content).
	// The destination path can be either a default location or any path specified by the user.
	// Syntax: [configmap|secret]:name[/key][@path], where name represents the resource name, key optionally represents the resource key to be filtered and path represents the destination path
	Resources []string `property:"resources" json:"resources,omitempty"`
	// A list of Persistent Volume Claims to be mounted. Syntax: [pvcname:/container/path]
	Volumes []string `property:"volumes" json:"volumes,omitempty"`
}
