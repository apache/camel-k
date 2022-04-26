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

// The platform trait is a base trait that is used to assign an integration platform to an integration.
//
// In case the platform is missing, the trait is allowed to create a default platform.
// This feature is especially useful in contexts where there's no need to provide a custom configuration for the platform
// (e.g. on OpenShift the default settings work, since there's an embedded container image registry).
//
// +camel-k:trait=platform.
type PlatformTrait struct {
	Trait `property:",squash" json:",inline"`
	// To create a default (empty) platform when the platform is missing.
	CreateDefault *bool `property:"create-default" json:"createDefault,omitempty"`
	// Indicates if the platform should be created globally in the case of global operator (default true).
	Global *bool `property:"global" json:"global,omitempty"`
	// To automatically detect from the environment if a default platform can be created (it will be created on OpenShift only).
	Auto *bool `property:"auto" json:"auto,omitempty"`
}
