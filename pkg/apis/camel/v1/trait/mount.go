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
	PlatformBaseTrait `property:",squash" json:",inline"`
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
	// Enable "hot reload" when a secret/configmap mounted is edited (default `false`). The configmap/secret must be
	// marked with `camel.apache.org/integration` label to be taken in account.
	HotReload *bool `property:"hot-reload" json:"hotReload,omitempty"`
	// Deprecated: use camel.properties or include your properties in an explicit property file backed by a configmap or secret.
	// Let the operator to treat configmaps or secret as plain properties file with their key/value list
	// (ie .spec.data["camel.my-property"] = my-value) (default `true`).
	ConfigsAsPropertyFiles *bool `property:"configs-as-property-files" json:"configsAsPropertyFiles,omitempty"`
	// Include any property file (suffix `.properties`) listed in configmaps/secrets provided in the `configs`
	// parameter as a runtime property file (default `true`).
	ScanConfigsForProperties *bool `property:"configs-as-properties" json:"configsAsProperties,omitempty"`
	// Deprecated: include your properties in an explicit property file backed by a secret.
	// Let the operator to scan for secret labeled with `camel.apache.org/kamelet` and `camel.apache.org/kamelet.configuration`.
	// These secrets are mounted to the application and treated as plain properties file with their key/value list
	// (ie .spec.data["camel.my-property"] = my-value) (default `true`).
	ScanKameletsImplicitLabelSecrets *bool `property:"scan-kamelets-implicit-label-secrets" json:"scanKameletsImplicitLabelSecrets,omitempty"`
}
