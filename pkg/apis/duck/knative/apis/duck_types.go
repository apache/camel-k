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

package apis

// Status is the common status of the Knative resources.
type Status struct {
	// ObservedGeneration is the 'Generation' of the resource that was last processed by the controller.
	// +optional
	ObservedGeneration int64 `json:"observedGeneration,omitempty"`

	// Conditions the latest available observations of a resource's current state.
	// +optional
	Conditions Conditions `json:"conditions,omitempty"`

	// Annotations is additional Status fields for the resource to save some additional state.
	// +optional
	Annotations map[string]string `json:"annotations,omitempty"`
}

// AddressStatus is the status of an addressable resource.
type AddressStatus struct {
	// Address is a single Addressable address.
	// +optional
	Address *Addressable `json:"address,omitempty"`
}

// Addressable is a destination for message delivery.
type Addressable struct {
	// URL is the address URL.
	URL *URL `json:"url,omitempty"`
}

// Destination represents a target of an invocation over HTTP.
type Destination struct {
	// Ref points to an Addressable.
	// +optional
	Ref *KReference `json:"ref,omitempty"`

	// URI can be an absolute URL pointing to the target or a relative URI resolved against Ref.
	// +optional
	URI *URL `json:"uri,omitempty"`
}

// KReference contains enough information to refer to another object.
type KReference struct {
	// Kind of the referent.
	Kind string `json:"kind"`

	// Namespace of the referent.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the referent.
	Name string `json:"name"`

	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`
}

// SourceSpec is the minimum resource shape to adhere to the Source specification.
type SourceSpec struct {
	// Sink is a reference to an object that will resolve to a uri to use as the sink.
	Sink Destination `json:"sink,omitempty"`
}

// BindingSpec is the minimum resource shape to adhere to the Binding specification.
type BindingSpec struct {
	// Subject references the resource whose runtime contract should be augmented by the Binding.
	Subject Reference `json:"subject"`
}

// Reference identifies the subject of a Binding.
type Reference struct {
	// API version of the referent.
	// +optional
	APIVersion string `json:"apiVersion,omitempty"`

	// Kind of the referent.
	// +optional
	Kind string `json:"kind,omitempty"`

	// Namespace of the referent.
	// +optional
	Namespace string `json:"namespace,omitempty"`

	// Name of the referent.
	// +optional
	Name string `json:"name,omitempty"`
}
