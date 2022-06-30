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

// The pod trait allows the customization of the Integration pods.
// It applies the `PodSpecTemplate` struct contained in the Integration `.spec.podTemplate` field,
// into the Integration deployment Pods template, using strategic merge patch.
//
// This can be used to customize the container where Camel routes execute,
// by using the `integration` container name.
//
// +camel-k:trait=pod.
type PodTrait struct {
	Trait `property:",squash" json:",inline"`
}
