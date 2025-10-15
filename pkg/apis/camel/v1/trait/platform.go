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
// +camel-k:trait=platform.
// +camel-k:internal.
type PlatformTrait struct {
	PlatformBaseTrait `property:",squash" json:",inline"`

	// Deprecated: no longer in use.
	CreateDefault *bool `property:"create-default" json:"createDefault,omitempty"`
	// Deprecated: no longer in use.
	Global *bool `property:"global" json:"global,omitempty"`
	// Deprecated: no longer in use.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}
