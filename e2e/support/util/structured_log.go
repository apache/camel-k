// +build integration

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
	"encoding/json"
	"math"
	"strconv"
	"time"

	"go.uber.org/zap/zapcore"
)

type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(s []byte) (err error) {
	f, err := strconv.ParseFloat(string(s), 10)
	if err != nil {
		return err
	}
	ns := (f - math.Floor(f)) * 1000000000
	*t = Time{
		time.Unix(int64(f), int64(ns)),
	}
	return nil
}

type Phase struct {
	Name string
}

func (p *Phase) UnmarshalJSON(b []byte) error {
	if b[0] != '"' {
		var tmp int

		json.Unmarshal(b, &tmp)
		p.Name = strconv.Itoa(tmp)

		return nil
	}

	if err := json.Unmarshal(b, &p.Name); err != nil {
		return err
	}

	return nil
}

type LogEntry struct {
	// Zap
	Level      zapcore.Level `json:"level,omitempty"`
	Timestamp  Time          `json:"ts,omitempty"`
	LoggerName string        `json:"logger,omitempty"`
	Message    string        `json:"msg,omitempty"`
	// Controller runtime
	RequestNamespace string `json:"request-namespace,omitempty"`
	RequestName      string `json:"request-name,omitempty"`
	ApiVersion       string `json:"api-version,omitempty"`
	Kind             string `json:"kind,omitempty"`
	// Camel K
	Namespace string `json:"ns,omitempty"`
	Name      string `json:"name,omitempty"`
	Phase     Phase `json:"phase,omitempty"`
	PhaseFrom string `json:"phase-from,omitempty"`
	PhaseTo   string `json:"phase-to,omitempty"`
}