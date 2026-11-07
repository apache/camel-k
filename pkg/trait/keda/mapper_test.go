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

package keda

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/apache/camel-k/v2/pkg/util/source"
)

func TestParseComponentURI(t *testing.T) {
	tests := []struct {
		name      string
		uri       string
		scheme    string
		pathValue string
		params    map[string]string
	}{
		{
			name:      "kafka with params",
			uri:       "kafka:orders?brokers=localhost:9092&groupId=myGroup",
			scheme:    "kafka",
			pathValue: "orders",
			params:    map[string]string{"brokers": "localhost:9092", "groupId": "myGroup"},
		},
		{
			name:      "aws2-sqs",
			uri:       "aws2-sqs:myQueue?region=us-east-1",
			scheme:    "aws2-sqs",
			pathValue: "myQueue",
			params:    map[string]string{"region": "us-east-1"},
		},
		{
			name:      "timer unsupported",
			uri:       "timer:tick?period=1000",
			scheme:    "timer",
			pathValue: "tick",
			params:    map[string]string{"period": "1000"},
		},
		{
			name:      "no query params",
			uri:       "direct:start",
			scheme:    "direct",
			pathValue: "start",
			params:    map[string]string{},
		},
		{
			name:      "empty uri",
			uri:       "",
			scheme:    "",
			pathValue: "",
			params:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			scheme, pathValue, params, err := ParseComponentURI(tt.uri)
			require.NoError(t, err)
			assert.Equal(t, tt.scheme, scheme)
			assert.Equal(t, tt.pathValue, pathValue)
			assert.Equal(t, tt.params, params)
		})
	}
}

func TestResolveBeanReference(t *testing.T) {
	bean := source.BeanDefinition{Name: "rabbitConnectionFactory"}
	tests := []struct {
		name      string
		reference string
		beans     []source.BeanDefinition
		found     bool
	}{
		{name: "direct reference", reference: "#rabbitConnectionFactory", beans: []source.BeanDefinition{bean}, found: true},
		{name: "missing bean", reference: "#missing", beans: []source.BeanDefinition{bean}},
		{name: "not a bean reference", reference: "rabbitConnectionFactory", beans: []source.BeanDefinition{bean}},
		{name: "dynamic reference", reference: "#${BEAN_NAME}", beans: []source.BeanDefinition{bean}},
		{name: "ambiguous bean name", reference: "#rabbitConnectionFactory", beans: []source.BeanDefinition{bean, bean}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolved, found := ResolveBeanReference(tt.reference, tt.beans)
			assert.Equal(t, tt.found, found)
			if tt.found {
				assert.Equal(t, bean, resolved)
			}
		})
	}
}
