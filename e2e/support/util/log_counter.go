//go:build integration
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
	"github.com/onsi/gomega/types"
)

type LogCounter struct {
	logs *[]LogEntry
}

func NewLogCounter(logs *[]LogEntry) *LogCounter {
	counter := LogCounter{
		logs: logs,
	}
	return &counter
}

func (w *LogCounter) Count(matcher types.GomegaMatcher) (uint, error) {
	count := uint(0)
	for _, log := range *w.logs {
		if match, err := matcher.Match(log); err != nil {
			return 0, err
		} else if match {
			count++
		}
	}
	return count, nil
}
