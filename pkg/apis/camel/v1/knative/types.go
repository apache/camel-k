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

import "fmt"

// CamelEnvironment is the top level configuration object expected by the Camel Knative component
type CamelEnvironment struct {
	Services []CamelServiceDefinition `json:"services"`
}

// NewCamelEnvironment creates a new env
func NewCamelEnvironment() CamelEnvironment {
	return CamelEnvironment{
		Services: make([]CamelServiceDefinition, 0),
	}
}

// CamelServiceDefinition defines the parameters to connect to Knative service. It's also used for exposed services
type CamelServiceDefinition struct {
	ServiceType CamelServiceType `json:"type"`
	Name        string           `json:"name"`
	// Deprecated: use URL instead
	Host string `json:"host,omitempty"`
	// Deprecated: use URL instead
	Port     *int              `json:"port,omitempty"`
	URL      string            `json:"url,omitempty"`
	Path     string            `json:"path,omitempty"`
	Metadata map[string]string `json:"metadata,omitempty"`
}

// CamelEndpointKind --
type CamelEndpointKind string

const (
	// CamelEndpointKindSource is a service that can be used to consume events
	CamelEndpointKindSource CamelEndpointKind = "source"
	// CamelEndpointKindSink is a service that can be used to send events to
	CamelEndpointKindSink CamelEndpointKind = "sink"
)

// CamelServiceType --
type CamelServiceType string

const (
	// CamelServiceTypeEndpoint is a callable endpoint
	CamelServiceTypeEndpoint CamelServiceType = "endpoint"
	// CamelServiceTypeChannel is a callable endpoint that will be also associated to a subscription
	CamelServiceTypeChannel CamelServiceType = "channel"
	// CamelServiceTypeEvent is used when the target service is the Knative broker
	CamelServiceTypeEvent CamelServiceType = "event"
)

func (s CamelServiceType) ResourceDescription(subject string) string {
	prefix := ""
	if s == CamelServiceTypeEvent {
		prefix = "broker for "
	}
	return fmt.Sprintf("%s%s %s", prefix, string(s), subject)
}

// Meta Options
const (
	CamelMetaKnativeKind       = "knative.kind"
	CamelMetaKnativeAPIVersion = "knative.apiVersion"
	CamelMetaKnativeReply      = "knative.reply"

	CamelMetaEndpointKind = "camel.endpoint.kind"

	CamelMetaFilterPrefix = "filter."
)
