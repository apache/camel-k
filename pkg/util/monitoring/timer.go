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

package monitoring

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

// Timer is a helper type to time functions. Use NewTimer to create new
// instances.
type Timer struct {
	begin time.Time
}

// NewTimer creates a new Timer.
func NewTimer() *Timer {
	return &Timer{
		begin: time.Now(),
	}
}

// ObserveDurationInSeconds records the duration passed since the Timer was created
// with NewTimer. It calls the Observe method of the provided Observer. The observed
// duration is also returned.
func (t *Timer) ObserveDurationInSeconds(o prometheus.Observer) time.Duration {
	d := time.Since(t.begin)
	o.Observe(d.Seconds())

	return d
}
