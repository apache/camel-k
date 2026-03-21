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

// The Gateway trait can be used to expose the service associated with the Integration
// to the outside world with a Kubernetes Gateway API. The trait is in charge to automatically discover associate the
// Integration Service generated with a Gateway and an HTTPRoute resource (HTTP/HTTPS protocol only supported).
//
// NOTE: if any other protocol is required, please create a request in order to develop it.
//
// +camel-k:trait=gateway.
//
//nolint:godoclint
type GatewayTrait struct {
	Trait `json:",inline" property:",squash"`

	// The class name to use for the gateway configuration.
	ClassName string `json:"className,omitempty" property:"class-name"`
	// The listeners in the format "port;protocol" (default, "8080;HTTP").
	Listeners []string `json:"listeners,omitempty" property:"listeners"`
}
