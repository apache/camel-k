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
// +camel-k:trait=mount.
type MountTrait struct {
	PlatformBaseTrait `json:",inline" property:",squash"`

	// A list of configuration pointing to configmap/secret.
	// The configuration are expected to be UTF-8 resources as they are processed by runtime Camel Context and tried to be parsed as property files.
	// They are also made available on the classpath in order to ease their usage directly from the Route.
	// Syntax: [configmap|secret]:name[/key], where name represents the resource name and key optionally represents the resource key to be filtered
	Configs []string `json:"configs,omitempty" property:"configs"`
	// A list of resources (text or binary content) pointing to configmap/secret.
	// The resources are expected to be any resource type (text or binary content).
	// The destination path can be either a default location or any path specified by the user.
	// Syntax: [configmap|secret]:name[/key][@path], where name represents the resource name, key optionally represents the resource key to be filtered and path represents the destination path
	Resources []string `json:"resources,omitempty" property:"resources"`
	// A list of Persistent Volume Claims to be mounted. Syntax: [pvcname:/container/path]. If the PVC is not found, the Integration fails.
	// You can use the syntax [pvcname:/container/path:size:accessMode<:storageClass>] to create a dynamic PVC based on the Storage Class provided
	// or the default cluster Storage Class. However, if the PVC exists, the operator would mount it.
	Volumes []string `json:"volumes,omitempty" property:"volumes"`
	// A list of EmptyDir volumes to be mounted. An optional size limit may be configured (default 500Mi).
	// Syntax: name:/container/path[:sizeLimit]
	EmptyDirs []string `json:"emptyDirs,omitempty" property:"empty-dirs"`
	// Enable "hot reload" when a secret/configmap mounted is edited (default `false`). The configmap/secret must be
	// marked with `camel.apache.org/integration` label to be taken in account. The resource will be watched for any kind change, also for
	// changes in metadata.
	HotReload *bool `json:"hotReload,omitempty" property:"hot-reload"`
	// Deprecated: no longer available since version 2.5.
	ScanKameletsImplicitLabelSecrets *bool `json:"scanKameletsImplicitLabelSecrets,omitempty" property:"scan-kamelets-implicit-label-secrets"`
}
