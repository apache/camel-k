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

// Allows constraining which nodes the integration pod(s) are eligible to be scheduled on, based on labels on the node,
// or with inter-pod affinity and anti-affinity, based on labels on pods that are already running on the nodes.
//
// It's disabled by default.
//
// +camel-k:trait=affinity.
type AffinityTrait struct {
	Trait `property:",squash" json:",inline"`
	// Always co-locates multiple replicas of the integration in the same node (default *false*).
	PodAffinity *bool `property:"pod-affinity" json:"podAffinity,omitempty"`
	// Never co-locates multiple replicas of the integration in the same node (default *false*).
	PodAntiAffinity *bool `property:"pod-anti-affinity" json:"podAntiAffinity,omitempty"`
	// Defines a set of nodes the integration pod(s) are eligible to be scheduled on, based on labels on the node.
	NodeAffinityLabels []string `property:"node-affinity-labels" json:"nodeAffinityLabels,omitempty"`
	// Defines a set of pods (namely those matching the label selector, relative to the given namespace) that the
	// integration pod(s) should be co-located with.
	PodAffinityLabels []string `property:"pod-affinity-labels" json:"podAffinityLabels,omitempty"`
	// Defines a set of pods (namely those matching the label selector, relative to the given namespace) that the
	// integration pod(s) should not be co-located with.
	PodAntiAffinityLabels []string `property:"pod-anti-affinity-labels" json:"podAntiAffinityLabels,omitempty"`
}
