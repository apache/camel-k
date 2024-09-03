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

// The 3scale trait can be used to automatically create annotations that allow
// 3scale to discover the generated service and make it available for API management.
//
// The 3scale trait is disabled by default.
//
// +camel-k:trait=3scale.
type ThreeScaleTrait struct {
	Trait `property:",squash" json:",inline"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// The scheme to use to contact the service (default `http`)
	Scheme string `property:"scheme" json:"scheme,omitempty"`
	// The path where the API is published (default `/`)
	Path string `property:"path" json:"path,omitempty"`
	// The port where the service is exposed (default `80`)
	Port int `property:"port" json:"port,omitempty"`
	// The path where the Open-API specification is published (default `/openapi.json`)
	DescriptionPath *string `property:"description-path" json:"descriptionPath,omitempty"`
}
