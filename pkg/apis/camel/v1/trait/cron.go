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

// The Cron trait can be used to customize the behaviour of periodic timer/cron based integrations.
//
// While normally an integration requires a pod to be always up and running, some periodic tasks, such as batch jobs,
// require to be activated at specific hours of the day or with a periodic delay of minutes.
// For such tasks, the cron trait can materialize the integration as a Kubernetes CronJob instead of a standard deployment,
// in order to save resources when the integration does not need to be executed.
//
// Integrations that start from the following components are evaluated by the cron trait: `timer`, `cron`, `quartz`.
//
// The rules for using a Kubernetes CronJob are the following:
// - `timer`: when periods can be written as cron expressions. E.g. `timer:tick?period=60000`.
// - `cron`, `quartz`: when the cron expression does not contain seconds (or the "seconds" part is set to 0). E.g.
//   `cron:tab?schedule=0/2${plus}*{plus}*{plus}*{plus}?` or `quartz:trigger?cron=0{plus}0/2{plus}*{plus}*{plus}*{plus}?`.
//
// +camel-k:trait=cron.
type CronTrait struct {
	Trait `property:",squash" json:",inline"`
	// The CronJob schedule for the whole integration. If multiple routes are declared, they must have the same schedule for this
	// mechanism to work correctly.
	Schedule string `property:"schedule" json:"schedule,omitempty"`
	// A comma separated list of the Camel components that need to be customized in order for them to work when the schedule is triggered externally by Kubernetes.
	// A specific customizer is activated for each specified component. E.g. for the `timer` component, the `cron-timer` customizer is
	// activated (it's present in the `org.apache.camel.k:camel-k-cron` library).
	//
	// Supported components are currently: `cron`, `timer` and `quartz`.
	Components string `property:"components" json:"components,omitempty"`
	// Use the default Camel implementation of the `cron` endpoint (`quartz`) instead of trying to materialize the integration
	// as Kubernetes CronJob.
	Fallback *bool `property:"fallback" json:"fallback,omitempty"`
	// Specifies how to treat concurrent executions of a Job.
	// Valid values are:
	// - "Allow": allows CronJobs to run concurrently;
	// - "Forbid" (default): forbids concurrent runs, skipping next run if previous run hasn't finished yet;
	// - "Replace": cancels currently running job and replaces it with a new one
	ConcurrencyPolicy string `property:"concurrency-policy" json:"concurrencyPolicy,omitempty"`
	// Automatically deploy the integration as CronJob when all routes are
	// either starting from a periodic consumer (only `cron`, `timer` and `quartz` are supported) or a passive consumer (e.g. `direct` is a passive consumer).
	//
	// It's required that all periodic consumers have the same period and it can be expressed as cron schedule (e.g. `1m` can be expressed as `0/1 * * * *`,
	// while `35m` or `50s` cannot).
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Optional deadline in seconds for starting the job if it misses scheduled
	// time for any reason.  Missed jobs executions will be counted as failed ones.
	StartingDeadlineSeconds *int64 `property:"starting-deadline-seconds" json:"startingDeadlineSeconds,omitempty"`
	// Specifies the duration in seconds, relative to the start time, that the job
	// may be continuously active before it is considered to be failed.
	// It defaults to 60s.
	ActiveDeadlineSeconds *int64 `property:"active-deadline-seconds" json:"activeDeadlineSeconds,omitempty"`
	// Specifies the number of retries before marking the job failed.
	// It defaults to 2.
	BackoffLimit *int32 `property:"backoff-limit" json:"backoffLimit,omitempty"`
}
