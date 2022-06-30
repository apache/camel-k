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

package v1

// The PDB trait allows to configure the PodDisruptionBudget resource for the Integration pods.
//
// +camel-k:trait=pdb.
type PDBTrait struct {
	Trait `property:",squash" json:",inline"`
	// The number of pods for the Integration that must still be available after an eviction.
	// It can be either an absolute number or a percentage.
	// Only one of `min-available` and `max-unavailable` can be specified.
	MinAvailable string `property:"min-available" json:"minAvailable,omitempty"`
	// The number of pods for the Integration that can be unavailable after an eviction.
	// It can be either an absolute number or a percentage (default `1` if `min-available` is also not set).
	// Only one of `max-unavailable` and `min-available` can be specified.
	MaxUnavailable string `property:"max-unavailable" json:"maxUnavailable,omitempty"`
}
