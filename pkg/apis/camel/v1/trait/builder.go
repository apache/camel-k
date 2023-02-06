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
	Trait `property:",squash" json:",inline"`
	// Enable verbose logging on build components that support it (e.g. Kaniko build pod).
	Verbose *bool `property:"verbose" json:"verbose,omitempty"`
	// A list of properties to be provided to the build task
	Properties []string `property:"properties" json:"properties,omitempty"`
}
