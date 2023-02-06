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

// The Owner trait ensures that all created resources belong to the integration being created
// and transfers annotations and labels on the integration onto these owned resources.
//
// +camel-k:trait=owner.
type OwnerTrait struct {
	Trait `property:",squash" json:",inline"`
	// The set of annotations to be transferred
	TargetAnnotations []string `property:"target-annotations" json:"targetAnnotations,omitempty"`
	// The set of labels to be transferred
	TargetLabels []string `property:"target-labels" json:"targetLabels,omitempty"`
}
