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
			uri:       "timer:tick?delay=123&period=3s",
			param:     "period",
			expected:  "3s",
			component: "timer",
		},
		{
			uri:       "timer:tick?period=3s&delay=4s",
			param:     "period",
			expected:  "3s",
			component: "timer",
		},
		{
			uri:       "timer:tick?delay=123&period=3s",
			param:     ".*",
			expected:  "",
			component: "timer",
		},
		{
			uri:       "timer:tick?delay=123&period=3s",
			param:     ".",
			expected:  "",
			component: "timer",
		},
		{
			uri:       "tick&delay=123&period=3s",
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
