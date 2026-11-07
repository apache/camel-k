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

import (
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// Status is the common status of the Knative resources.
type Status struct {
	// Conditions the latest available observations of a resource's current state.
	// The Knative conditions are read as Kubernetes API conditions, as they share
	// the fields Camel K uses (type, status, reason and message).
	// +optional
	Conditions []metav1.Condition `json:"conditions,omitempty"`
}

// Addressable is a destination for message delivery.
type Addressable struct {
	// URL is the address URL.
	URL *URL `json:"url,omitempty"`
}
