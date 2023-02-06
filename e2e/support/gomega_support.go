//go:build integration
// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package support

import (
	"fmt"
	"os"
	"time"

	. "github.com/onsi/gomega"
)

const (
	eventuallyTimeoutEnvVarName           = "E2E_DEFAULT_EVENTUALLY_TIMEOUT"
	eventuallyPollingIntervalEnvVarName   = "E2E_DEFAULT_EVENTUALLY_POLLING_INTERVAL"
	consistentlyDurationEnvVarName        = "E2E_DEFAULT_CONSISTENTLY_DURATION"
	consistentlyPollingIntervalEnvVarName = "E2E_DEFAULT_CONSISTENTLY_POLLING_INTERVAL"
)

func init() {
	SetDefaultEventuallyTimeout(durationFromEnv(eventuallyTimeoutEnvVarName, time.Second))
	SetDefaultEventuallyPollingInterval(durationFromEnv(eventuallyPollingIntervalEnvVarName, 500*time.Millisecond))
	SetDefaultConsistentlyDuration(durationFromEnv(consistentlyDurationEnvVarName, 100*time.Millisecond))
	SetDefaultConsistentlyPollingInterval(durationFromEnv(consistentlyPollingIntervalEnvVarName, 500*time.Millisecond))
}

func durationFromEnv(key string, defaultDuration time.Duration) time.Duration {
	value := os.Getenv(key)
	if value == "" {
		return defaultDuration
	}
	duration, err := time.ParseDuration(value)
	if err != nil {
		panic(fmt.Sprintf("Expected a duration when using %s!  Parse error %v", key, err))
	}
	return duration
}
