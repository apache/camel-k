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

var LogEntryNoop = func(*LogEntry) {}

type LogWalker struct {
	logs  *[]LogEntry
	steps []logWalkerStep
}

type logWalkerStep struct {
	matcher  types.GomegaMatcher
	consumer func(*LogEntry)
}

func NewLogWalker(logs *[]LogEntry) *LogWalker {
	walker := LogWalker{
		logs: logs,
	}
	return &walker
}

func (w *LogWalker) AddStep(m types.GomegaMatcher, c func(*LogEntry)) *LogWalker {
	w.steps = append(w.steps, logWalkerStep{
		matcher:  m,
		consumer: c,
	})
	return w
}

func (w *LogWalker) Walk() error {
	i := 0
	step := w.steps[i]
	for _, log := range *w.logs {
		if match, err := step.matcher.Match(log); err != nil {
			return err
		} else if match {
			step.consumer(&log)
			if i == len(w.steps)-1 {
				break
			}
			i++
			step = w.steps[i]
		}
	}
	return nil
}
