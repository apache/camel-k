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

// The Knative trait automatically discovers addresses of Knative resources and inject them into the
// running integration.
//
// The full Knative configuration is injected in the CAMEL_KNATIVE_CONFIGURATION in JSON format.
// The Camel Knative component will then use the full configuration to configure the routes.
//
// The trait is enabled by default when the Knative profile is active.
//
// +camel-k:trait=knative.
type KnativeTrait struct {
	Trait `property:",squash" json:",inline"`
	// Can be used to inject a Knative complete configuration in JSON format.
	Configuration string `property:"configuration" json:"config,omitempty"`
	// List of channels used as source of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSources []string `property:"channel-sources" json:"channelSources,omitempty"`
	// List of channels used as destination of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSinks []string `property:"channel-sinks" json:"channelSinks,omitempty"`
	// List of channels used as source of integration routes.
	EndpointSources []string `property:"endpoint-sources" json:"endpointSources,omitempty"`
	// List of endpoints used as destination of integration routes.
	// Can contain simple endpoint names or full Camel URIs.
	EndpointSinks []string `property:"endpoint-sinks" json:"endpointSinks,omitempty"`
	// List of event types that the integration will be subscribed to.
	// Can contain simple event types or full Camel URIs (to use a specific broker different from "default").
	EventSources []string `property:"event-sources" json:"eventSources,omitempty"`
	// List of event types that the integration will produce.
	// Can contain simple event types or full Camel URIs (to use a specific broker).
	EventSinks []string `property:"event-sinks" json:"eventSinks,omitempty"`
	// Enables filtering on events based on the header "ce-knativehistory". Since this header has been removed in newer versions of
	// Knative, filtering is disabled by default.
	FilterSourceChannels *bool `property:"filter-source-channels" json:"filterSourceChannels,omitempty"`
	// Allows binding the integration to a sink via a Knative SinkBinding resource.
	// This can be used when the integration targets a single sink.
	// It's enabled by default when the integration targets a single sink
	// (except when the integration is owned by a Knative source).
	SinkBinding *bool `property:"sink-binding" json:"sinkBinding,omitempty"`
	// Enable automatic discovery of all trait properties.
	Auto *bool `property:"auto" json:"auto,omitempty"`
}
