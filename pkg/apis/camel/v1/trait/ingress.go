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

// The Ingress trait can be used to expose the service associated with the integration
// to the outside world with a Kubernetes Ingress.
//
// It's enabled by default whenever a Service is added to the integration (through the `service` trait).
//
// +camel-k:trait=ingress.
type IngressTrait struct {
	Trait `property:",squash" json:",inline"`
	// **Required**. To configure the host exposed by the ingress.
	Host string `property:"host" json:"host,omitempty"`
	// To automatically add an ingress whenever the integration uses a HTTP endpoint consumer.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}
