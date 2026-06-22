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

package platform

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

// setEnv sets (or, when value is nil, unsets) an environment variable for the duration of the test.
func setEnv(t *testing.T, key string, value *string) {
	t.Helper()
	orig, had := os.LookupEnv(key)
	t.Cleanup(func() {
		if had {
			_ = os.Setenv(key, orig)
		} else {
			_ = os.Unsetenv(key)
		}
	})
	if value == nil {
		_ = os.Unsetenv(key)
	} else {
		_ = os.Setenv(key, *value)
	}
}

func ptr(s string) *string { return &s }

func TestIsCurrentOperatorGlobal(t *testing.T) {
	tests := []struct {
		name     string
		watch    *string
		selector *string
		global   bool
	}{
		{name: "unset watch, no selector -> global", watch: nil, selector: nil, global: true},
		{name: "empty watch, no selector -> global", watch: ptr(""), selector: nil, global: true},
		{name: "blank watch, no selector -> global", watch: ptr("  "), selector: nil, global: true},
		{name: "single namespace -> local", watch: ptr("ns1"), selector: nil, global: false},
		{name: "multi namespace -> local", watch: ptr("ns1,ns2"), selector: nil, global: false},
		{name: "selector only -> local", watch: ptr(""), selector: ptr("camel-k-enabled=true"), global: false},
		{name: "blank selector behaves as unset -> global", watch: ptr(""), selector: ptr("  "), global: true},
		{name: "selector with list -> local", watch: ptr("ns1"), selector: ptr("a=b"), global: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, OperatorWatchNamespaceEnvVariable, tc.watch)
			setEnv(t, OperatorWatchNamespaceSelectorEnvVariable, tc.selector)
			assert.Equal(t, tc.global, IsCurrentOperatorGlobal())
		})
	}
}

func TestGetWatchNamespaces(t *testing.T) {
	tests := []struct {
		name     string
		watch    *string
		expected []string
	}{
		{name: "unset -> nil", watch: nil, expected: nil},
		{name: "empty -> nil", watch: ptr(""), expected: nil},
		{name: "single", watch: ptr("ns1"), expected: []string{"ns1"}},
		{name: "list", watch: ptr("ns1,ns2,ns3"), expected: []string{"ns1", "ns2", "ns3"}},
		{name: "trims whitespace", watch: ptr(" ns1 , ns2 "), expected: []string{"ns1", "ns2"}},
		{name: "drops empties", watch: ptr("ns1,,ns2,"), expected: []string{"ns1", "ns2"}},
		{name: "de-duplicates preserving order", watch: ptr("ns1,ns2,ns1"), expected: []string{"ns1", "ns2"}},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			setEnv(t, OperatorWatchNamespaceEnvVariable, tc.watch)
			assert.Equal(t, tc.expected, GetWatchNamespaces())
		})
	}
}

func TestGetWatchNamespaceSelector(t *testing.T) {
	setEnv(t, OperatorWatchNamespaceSelectorEnvVariable, nil)
	assert.Equal(t, "", GetWatchNamespaceSelector())

	setEnv(t, OperatorWatchNamespaceSelectorEnvVariable, ptr("  camel-k-enabled=true  "))
	assert.Equal(t, "camel-k-enabled=true", GetWatchNamespaceSelector())
}
