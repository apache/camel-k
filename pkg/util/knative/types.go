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

package knative

// CamelKnativeEnvironment is the top level configuration object expected by the Camel Knative component
type CamelKnativeEnvironment struct {
	Services []CamelKnativeServiceDefinition `json:"services"`
}

// NewCamelKnativeEnvironment creates a new env
func NewCamelKnativeEnvironment() CamelKnativeEnvironment {
	return CamelKnativeEnvironment{
		Services: make([]CamelKnativeServiceDefinition, 0),
	}
}

// CamelKnativeServiceDefinition defines the parameters to connect to Knative service. It's also used for exposed services
type CamelKnativeServiceDefinition struct {
	ServiceType CamelKnativeServiceType `json:"type"`
	Protocol    CamelKnativeProtocol    `json:"protocol"`
	Name        string                  `json:"name"`
	Host        string                  `json:"host"`
	Port        int                     `json:"port"`
	Metadata    map[string]string       `json:"metadata"`
}

// CamelKnativeServiceType --
type CamelKnativeServiceType string

const (
	// CamelKnativeServiceTypeEndpoint is a callable endpoint
	CamelKnativeServiceTypeEndpoint CamelKnativeServiceType = "endpoint"
	// CamelKnativeServiceTypeChannel is a callable endpoint that will be also associated to a subscription
	CamelKnativeServiceTypeChannel CamelKnativeServiceType = "channel"
)

// CamelKnativeProtocol is the communication protocol to use for the service
type CamelKnativeProtocol string

// Knative protocols
const (
	CamelKnativeProtocolHTTP  CamelKnativeProtocol = "http"
	CamelKnativeProtocolHTTPS CamelKnativeProtocol = "https"
)

// Meta Options
const (
	CamelKnativeMetaServicePath     = "service.path"
	CamelKnativeMetaServiceID       = "service.id"
	CamelKnativeMetaServiceName     = "service.name"
	CamelKnativeMetaServiceHost     = "service.host"
	CamelKnativeMetaServicePort     = "service.port"
	CamelKnativeMetaServiceZone     = "service.zone"
	CamelKnativeMetaServiceProtocol = "service.protocol"
)
