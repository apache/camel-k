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

package uri

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestQueryParameter(t *testing.T) {
	tests := []struct {
		uri       string
		param     string
		expected  string
		component string
	}{
		{
			uri:       "direct:endpoint",
			param:     "notPresent",
			expected:  "",
			component: "direct",
		},
		{
			uri:       "knative:endpoint/channel?brokerURI=123",
			param:     "brokeruri",
			expected:  "",
			component: "knative",
		},
		{
			uri:       "knative:endpoint/channel?brokerURI=123",
			param:     "brokerURI",
			expected:  "123",
			component: "knative",
		},
		{
			uri:       "timer:tick?delay=123&period=3000",
			param:     "period",
			expected:  "3000",
			component: "timer",
		},
		{
			uri:       "timer:tick?period=3000&delay=4000",
			param:     "period",
			expected:  "3000",
			component: "timer",
		},
		{
			uri:       "timer:tick?delay=123&period=3000",
			param:     ".*",
			expected:  "",
			component: "timer",
		},
		{
			uri:       "timer:tick?delay=123&period=3000",
			param:     ".",
			expected:  "",
			component: "timer",
		},
		{
			uri:       "tick&delay=123&period=3000",
			param:     "delay",
			expected:  "",
			component: "",
		},
	}

	for _, test := range tests {
		thetest := test
		t.Run(thetest.uri, func(t *testing.T) {
			param := GetQueryParameter(thetest.uri, thetest.param)
			assert.Equal(t, thetest.expected, param)
		})
	}
}

func TestAppendParameters(t *testing.T) {
	tests := []struct {
		prefix   string
		params   map[string]string
		expected string
	}{
		{
			prefix:   "kamelet://mykamelet",
			params:   nil,
			expected: "kamelet://mykamelet",
		},
		{
			prefix: "kamelet://mykamelet",
			params: map[string]string{
				"a": "b",
			},
			expected: "kamelet://mykamelet?a=b",
		},
		{
			prefix: "kamelet://mykamelet",
			params: map[string]string{
				"a": "b",
				"c": "d",
			},
			expected: "kamelet://mykamelet?a=b&c=d",
		},
		{
			prefix: "kamelet://mykamelet",
			params: map[string]string{
				"z": "y",
				"c": "d",
			},
			expected: "kamelet://mykamelet?c=d&z=y",
		},
		{
			prefix: "kamelet://mykamelet?h=m",
			params: map[string]string{
				"z": "y",
				"c": "d",
			},
			expected: "kamelet://mykamelet?h=m&c=d&z=y",
		},
	}

	for i := range tests {
		test := tests[i]
		t.Run(fmt.Sprintf("appendParameters-%d-%s", i, test.expected), func(t *testing.T) {
			uri := AppendParameters(test.prefix, test.params)
			assert.Equal(t, test.expected, uri)
		})
	}
}

func TestCamelURIFormat(t *testing.T) {
	tests := []struct {
		uri     string
		invalid bool
	}{
		{
			uri: "knative:channnel",
		},
		{
			uri: "atomix-value:",
		},
		{
			uri: "aws-ec2:",
		},
		{
			uri: "coap+tcp:",
		},
		{
			uri: "solrCloud:",
		},
		{
			uri:     "PostgreSQL:db",
			invalid: true,
		},
		{
			uri:     "postgres.org/v1alpha1:PostgreSQL:db",
			invalid: true,
		},
	}

	for i := range tests {
		tc := tests[i]
		t.Run(fmt.Sprintf("%d-%s", i, tc.uri), func(t *testing.T) {
			assert.Equal(t, !tc.invalid, HasCamelURIFormat(tc.uri))
		})
	}
}
