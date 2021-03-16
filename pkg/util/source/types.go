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

package source

import "github.com/scylladb/go-set/strset"

// Metadata --
type Metadata struct {
	// All starting URIs of defined routes
	FromURIs []string
	// All end URIs of defined routes
	ToURIs []string
	// All error handlers URIs of defined routes
	ErrorHandlerURIs []string
	// All inferred dependencies required to run the integration
	Dependencies *strset.Set
	// ExposesHTTPServices indicates if a route defined by the source is exposed
	// through HTTP
	ExposesHTTPServices bool
	// PassiveEndpoints indicates that the source contains only passive endpoints that
	// are activated from external calls, including HTTP (useful to determine if the
	// integration can scale to 0)
	PassiveEndpoints bool
	// RequiredCapabilities lists the capabilities required by the integration
	// to run
	RequiredCapabilities *strset.Set
}

// NewMetadata --
func NewMetadata() Metadata {
	return Metadata{
		FromURIs:             make([]string, 0),
		ToURIs:               make([]string, 0),
		Dependencies:         strset.New(),
		RequiredCapabilities: strset.New(),
	}
}
