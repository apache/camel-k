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

// The GC Trait garbage-collects all resources that are no longer necessary upon integration updates.
//
// +camel-k:trait=gc.
type GCTrait struct {
	Trait `property:",squash" json:",inline"`
	// Discovery client cache to be used, either `disabled`, `disk` or `memory` (default `memory`).
	// Deprecated: to be removed from trait configuration.
	DiscoveryCache *DiscoveryCacheType `property:"discovery-cache" json:"discoveryCache,omitempty"`
}

// +kubebuilder:validation:Enum=disabled;disk;memory
type DiscoveryCacheType string

const (
	DisabledDiscoveryCache DiscoveryCacheType = "disabled"
	DiskDiscoveryCache     DiscoveryCacheType = "disk"
	MemoryDiscoveryCache   DiscoveryCacheType = "memory"
)
