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

// The Prometheus trait configures a Prometheus-compatible endpoint. It also creates a `PodMonitor` resource,
// so that the endpoint can be scraped automatically, when using the Prometheus operator.
//
// The metrics are exposed using MicroProfile Metrics.
//
// WARNING: The creation of the `PodMonitor` resource requires the https://github.com/coreos/prometheus-operator[Prometheus Operator]
// custom resource definition to be installed.
// You can set `pod-monitor` to `false` for the Prometheus trait to work without the Prometheus Operator.
//
// The Prometheus trait is disabled by default.
//
// +camel-k:trait=prometheus.
type PrometheusTrait struct {
	Trait `property:",squash" json:",inline"`
	// Whether a `PodMonitor` resource is created (default `true`).
	PodMonitor *bool `property:"pod-monitor" json:"podMonitor,omitempty"`
	// The `PodMonitor` resource labels, applicable when `pod-monitor` is `true`.
	PodMonitorLabels []string `property:"pod-monitor-labels" json:"podMonitorLabels,omitempty"`
}
