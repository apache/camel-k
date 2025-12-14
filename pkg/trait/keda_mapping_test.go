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

package trait

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
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
			name:      "spring-rabbitmq",
			uri:       "spring-rabbitmq:exchange?queues=myQueue&addresses=localhost:5672",
			scheme:    "spring-rabbitmq",
			pathValue: "exchange",
			params:    map[string]string{"queues": "myQueue", "addresses": "localhost:5672"},
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
			scheme, pathValue, params, err := parseComponentURI(tt.uri)
			require.NoError(t, err)
			assert.Equal(t, tt.scheme, scheme)
			assert.Equal(t, tt.pathValue, pathValue)
			assert.Equal(t, tt.params, params)
		})
	}
}

func TestMapToKedaTrigger(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		expectNil    bool
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "kafka trigger",
			uri:          "kafka:orders?brokers=broker:9092&groupId=grp",
			expectNil:    false,
			expectedType: "kafka",
			expectedMeta: map[string]string{
				"topic":            "orders",
				"bootstrapServers": "broker:9092",
				"consumerGroup":    "grp",
			},
		},
		{
			name:         "aws2-sqs trigger",
			uri:          "aws2-sqs:myQueue?region=us-east-1",
			expectNil:    false,
			expectedType: "aws-sqs-queue",
			expectedMeta: map[string]string{
				"queueURL":  "myQueue",
				"awsRegion": "us-east-1",
			},
		},
		{
			name:         "spring-rabbitmq trigger",
			uri:          "spring-rabbitmq:exchange?queues=myQueue&addresses=localhost:5672",
			expectNil:    false,
			expectedType: "rabbitmq",
			expectedMeta: map[string]string{
				"queueName": "myQueue",
				"host":      "localhost:5672",
			},
		},
		{
			name:      "timer no trigger",
			uri:       "timer:tick",
			expectNil: true,
		},
		{
			name:      "direct no trigger",
			uri:       "direct:start",
			expectNil: true,
		},
		{
			name:      "empty uri no trigger",
			uri:       "",
			expectNil: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := mapToKedaTrigger(tt.uri)
			require.NoError(t, err)
			if tt.expectNil {
				assert.Nil(t, trigger)
			} else {
				require.NotNil(t, trigger)
				assert.Equal(t, tt.expectedType, trigger.Type)
				assert.Equal(t, tt.expectedMeta, trigger.Metadata)
			}
		})
	}
}
