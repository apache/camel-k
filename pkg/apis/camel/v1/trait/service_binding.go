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

// WARNING: This trait is no longer in use.
//

// The Service Binding trait allows users to connect to Services in Kubernetes:
// https://github.com/k8s-service-bindings/spec#service-binding
// As the specification is still evolving this is subject to change.
//
// +camel-k:trait=service-binding.
// +camel-k:internal.
type ServiceBindingTrait struct {
	Trait `property:",squash" json:",inline"`
	// List of Services in the form [[apigroup/]version:]kind:[namespace/]name
	Services []string `property:"services" json:"services,omitempty"`
}
