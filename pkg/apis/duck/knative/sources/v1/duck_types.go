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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/v2/pkg/apis/duck/knative/apis"
)

// +kubebuilder:object:root=true

// SinkBinding is a partial schema of the Knative Eventing SinkBinding resource: a
// Binding that resolves its sink to a URL and projects it into the subject.
type SinkBinding struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`

	Spec   SinkBindingSpec   `json:"spec"`
	Status SinkBindingStatus `json:"status"`
}

// SinkBindingSpec holds the desired state of the SinkBinding.
type SinkBindingSpec struct {
	// SourceSpec provides the Sink to resolve.
	apis.SourceSpec `json:",inline"`

	// BindingSpec provides the Subject to augment.
	apis.BindingSpec `json:",inline"`
}

// SinkBindingStatus communicates the observed state of the SinkBinding.
type SinkBindingStatus struct {
	apis.Status `json:",inline"`
}

// +kubebuilder:object:root=true

// SinkBindingList contains a list of SinkBinding.
type SinkBindingList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`

	Items []SinkBinding `json:"items"`
}
