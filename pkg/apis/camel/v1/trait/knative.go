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
// The Camel Knative component will then use the full configuration to configure the routes.
//
// The trait is enabled by default when the Knative profile is active.
//
// +camel-k:trait=knative.
//
//nolint:godoclint
type KnativeTrait struct {
	Trait `json:",inline" property:",squash"`

	// Can be used to inject a Knative complete configuration in JSON format.
	Configuration string `json:"config,omitempty" property:"configuration"`
	// List of channels used as source of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSources []string `json:"channelSources,omitempty" property:"channel-sources"`
	// List of channels used as destination of integration routes.
	// Can contain simple channel names or full Camel URIs.
	ChannelSinks []string `json:"channelSinks,omitempty" property:"channel-sinks"`
	// List of channels used as source of integration routes.
	EndpointSources []string `json:"endpointSources,omitempty" property:"endpoint-sources"`
	// List of endpoints used as destination of integration routes.
	// Can contain simple endpoint names or full Camel URIs.
	EndpointSinks []string `json:"endpointSinks,omitempty" property:"endpoint-sinks"`
	// List of event types that the integration will be subscribed to.
	// Can contain simple event types or full Camel URIs (to use a specific broker different from "default").
	EventSources []string `json:"eventSources,omitempty" property:"event-sources"`
	// List of event types that the integration will produce.
	// Can contain simple event types or full Camel URIs (to use a specific broker).
	EventSinks []string `json:"eventSinks,omitempty" property:"event-sinks"`
	// Enables filtering on events based on the header "ce-knativehistory". Since this header has been removed in newer versions of
	// Knative, filtering is disabled by default.
	FilterSourceChannels *bool `json:"filterSourceChannels,omitempty" property:"filter-source-channels"`
	// Allows binding the integration to a sink via a Knative SinkBinding resource.
	// This can be used when the integration targets a single sink.
	// It's enabled by default when the integration targets a single sink
	// (except when the integration is owned by a Knative source).
	SinkBinding *bool `json:"sinkBinding,omitempty" property:"sink-binding"`
	// Enable automatic discovery of all trait properties.
	Auto *bool `json:"auto,omitempty" property:"auto"`
	// Enables the camel-k-operator to set the "bindings.knative.dev/include=true" label to the namespace
	// As Knative requires this label to perform injection of K_SINK URL into the service.
	// If this is false, the integration pod may start and fail, read the SinkBinding Knative documentation. (default: true)
	NamespaceLabel *bool `json:"namespaceLabel,omitempty" property:"namespace-label"`
	// Sets filter attributes on the event stream (such as event type, source, subject and so on).
	// A list of key-value pairs that represent filter attributes and its values.
	// The syntax is KEY=VALUE, e.g., `source="my.source"`.
	// Filter attributes get set on the Knative trigger that is being created as part of this integration.
	Filters []string `json:"filters,omitempty" property:"filters"`
	// Enables the default filtering for the Knative trigger using the event type
	// If this is true, the created Knative trigger uses the event type as a filter on the event stream when no other filter criteria is given. (default: true)
	FilterEventType *bool `json:"filterEventType,omitempty" property:"filter-event-type"`
}
