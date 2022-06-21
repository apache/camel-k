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

// This trait sets Tolerations over Integration pods. Tolerations allow (but do not require) the pods to schedule onto nodes with matching taints.
// See https://kubernetes.io/docs/concepts/scheduling-eviction/taint-and-toleration/ for more details.
//
// The toleration should be expressed in a similar manner that of taints, i.e., `Key[=Value]:Effect[:Seconds]`, where values in square brackets are optional.
//
// For examples:
//
// - `node-role.kubernetes.io/master:NoSchedule`
// - `node.kubernetes.io/network-unavailable:NoExecute:3000`
// - `disktype=ssd:PreferNoSchedule`
//
// It's disabled by default.
//
// +camel-k:trait=toleration.
type TolerationTrait struct {
	Trait `property:",squash" json:",inline"`
	// The list of taints to tolerate, in the form `Key[=Value]:Effect[:Seconds]`
	Taints []string `property:"taints" json:"taints,omitempty"`
}
