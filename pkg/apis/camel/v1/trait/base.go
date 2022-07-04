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

// Base type for all traits.
type Trait struct {
	// Can be used to enable or disable a trait. All traits share this common property.
	Enabled *bool `property:"enabled" json:"enabled,omitempty"`

	// Legacy trait configuration parameters.
	// Deprecated: for backward compatibility.
	Configuration *Configuration `json:"configuration,omitempty"`
}

// Deprecated: for backward compatibility.
type Configuration struct {
	RawMessage `json:",inline"`
}

// +kubebuilder:validation:Type=object
// +kubebuilder:validation:Format=""
// +kubebuilder:pruning:PreserveUnknownFields
// Deprecated: for backward compatibility.
type RawMessage []byte
