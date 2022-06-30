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

// The Service trait exposes the integration with a Service resource so that it can be accessed by other applications
// (or integrations) in the same namespace.
//
// It's enabled by default if the integration depends on a Camel component that can expose a HTTP endpoint.
//
// +camel-k:trait=service.
type ServiceTrait struct {
	Trait `property:",squash" json:",inline"`
	// To automatically detect from the code if a Service needs to be created.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Enable Service to be exposed as NodePort (default `false`).
	NodePort *bool `property:"node-port" json:"nodePort,omitempty"`
}
