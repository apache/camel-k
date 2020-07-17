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

// The Prometheus trait configures a Prometheus-compatible endpoint. It also exposes the integration with a `Service`
// and a `ServiceMonitor` resources, so that the endpoint can be scraped automatically, when using the Prometheus
// operator.
//
// The metrics exposed vary depending on the configured runtime. With Quarkus, the metrics are exposed
// using MicroProfile Metrics. While with the default runtime, they are exposed using the Prometheus JMX exporter.
//
// WARNING: The creation of the `ServiceMonitor` resource requires the https://github.com/coreos/prometheus-operator[Prometheus Operator]
// custom resource definition to be installed.
// You can set `service-monitor` to `false` for the Prometheus trait to work without the Prometheus operator.
//
// It's disabled by default.
//
// +camel-k:trait=prometheus
type PrometheusTrait struct {
	Trait `property:",squash" json:",inline"`
	// The Prometheus endpoint port (default `9779`, or `8080` with Quarkus).
	Port *int `property:"port" json:"port,omitempty"`
	// Whether a `ServiceMonitor` resource is created (default `true`).
	ServiceMonitor bool `property:"service-monitor" json:"serviceMonitor,omitempty"`
	// The `ServiceMonitor` resource labels, applicable when `service-monitor` is `true`.
	ServiceMonitorLabels []string `property:"service-monitor-labels" json:"serviceMonitorLabels,omitempty"`
	// To use a custom ConfigMap containing the Prometheus JMX exporter configuration (under the `content` ConfigMap key).
	// When this property is left empty (default), Camel K generates a standard Prometheus configuration for the integration.
	// It is not applicable when using Quarkus.
	ConfigMap string `property:"configmap" json:"configMap,omitempty"`
}
