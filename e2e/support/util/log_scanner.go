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
	"bufio"
	"context"
	"fmt"
	"io"
	"strings"
)

// LogScanner can attach to a stream and check if a value is printed
type LogScanner struct {
	in     io.Reader
	ctx    context.Context
	values map[string]bool
}

// NewLogScanner --
func NewLogScanner(ctx context.Context, in io.Reader, values ...string) *LogScanner {
	scanner := LogScanner{
		ctx:    ctx,
		in:     in,
		values: make(map[string]bool),
	}
	for _, v := range values {
		scanner.values[v] = false
	}
	go scanner.startScan()
	return &scanner
}

func (s *LogScanner) startScan() {
	scanner := bufio.NewScanner(s.in)
	for scanner.Scan() {
		if s.ctx.Err() != nil {
			return
		}
		text := scanner.Text()
		fmt.Println(text)
		for k := range s.values {
			if strings.Contains(text, k) {
				fmt.Printf("LogScanner - Found match for: %s\n", k)
				s.values[k] = true
			}
		}
	}
}

// IsFound returns if the string has been found in the logs
func (s *LogScanner) IsFound(value string) func() bool {
	return func() bool {
		return s.values[value]
	}
}
