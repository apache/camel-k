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

// The OpenAPI DSL trait is internally used to allow creating integrations from a OpenAPI specs.
//
// WARNING: The Openapi trait is **deprecated** and will removed in future release versions:
// use Camel REST contract first instead, https://camel.apache.org/manual/rest-dsl-openapi.html
//
// +camel-k:trait=openapi.
// +camel-k:deprecated=2.5.0.
type OpenAPITrait struct {
	PlatformBaseTrait `property:",squash" json:",inline"`
	// The configmaps holding the spec of the OpenAPI (compatible with > 3.0 spec only).
	Configmaps []string `property:"configmaps" json:"configmaps,omitempty"`
}
