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

// The Knative Service trait allows configuring options when running the Integration as a Knative service, instead of
// a standard Kubernetes Deployment.
//
// Running an Integration as a Knative Service enables auto-scaling (and scaling-to-zero), but those features
// are only relevant when the Camel route(s) use(s) an HTTP endpoint consumer.
//
// +camel-k:trait=knative-service.
type KnativeServiceTrait struct {
	Trait `property:",squash" json:",inline"`
	// Configures the Knative autoscaling class property (e.g. to set `hpa.autoscaling.knative.dev` or `kpa.autoscaling.knative.dev` autoscaling).
	//
	// Refer to the Knative documentation for more information.
	Class string `property:"autoscaling-class" json:"class,omitempty"`
	// Configures the Knative autoscaling metric property (e.g. to set `concurrency` based or `cpu` based autoscaling).
	//
	// Refer to the Knative documentation for more information.
	Metric string `property:"autoscaling-metric" json:"autoscalingMetric,omitempty"`
	// Sets the allowed concurrency level or CPU percentage (depending on the autoscaling metric) for each Pod.
	//
	// Refer to the Knative documentation for more information.
	Target *int `property:"autoscaling-target" json:"autoscalingTarget,omitempty"`
	// The minimum number of Pods that should be running at any time for the integration. It's **zero** by default, meaning that
	// the integration is scaled down to zero when not used for a configured amount of time.
	//
	// Refer to the Knative documentation for more information.
	MinScale *int `property:"min-scale" json:"minScale,omitempty"`
	// An upper bound for the number of Pods that can be running in parallel for the integration.
	// Knative has its own cap value that depends on the installation.
	//
	// Refer to the Knative documentation for more information.
	MaxScale *int `property:"max-scale" json:"maxScale,omitempty"`
	// Enables to gradually shift traffic to the latest Revision and sets the rollout duration.
	// It's disabled by default and must be expressed as a Golang `time.Duration` string representation,
	// rounded to a second precision.
	RolloutDuration string `property:"rollout-duration" json:"rolloutDuration,omitempty"`
	// Automatically deploy the integration as Knative service when all conditions hold:
	//
	// * Integration is using the Knative profile
	// * All routes are either starting from a HTTP based consumer or a passive consumer (e.g. `direct` is a passive consumer)
	Auto *bool `property:"auto" json:"auto,omitempty"`
}
