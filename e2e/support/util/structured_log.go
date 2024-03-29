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

package util

import (
	"time"

	"go.uber.org/zap/zapcore"
)

type LogEntry struct {
	// Zap
	Level      zapcore.Level `json:"level,omitempty"`
	Timestamp  time.Time     `json:"ts,omitempty"`
	LoggerName string        `json:"logger,omitempty"`
	Message    string        `json:"msg,omitempty"`
	// Controller runtime
	RequestNamespace string `json:"request-namespace,omitempty"`
	RequestName      string `json:"request-name,omitempty"`
	APIVersion       string `json:"api-version,omitempty"`
	Kind             string `json:"kind,omitempty"`
	// Camel K
	Namespace string `json:"ns,omitempty"`
	Name      string `json:"name,omitempty"`
	Phase     string `json:"phase,omitempty"`
	PhaseFrom string `json:"phase-from,omitempty"`
	PhaseTo   string `json:"phase-to,omitempty"`
}
