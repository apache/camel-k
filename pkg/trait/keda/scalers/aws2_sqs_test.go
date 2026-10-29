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

package scalers

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	keda "github.com/apache/camel-k/v2/pkg/trait/keda"
)

func TestAWS2SQSScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("aws2-sqs")
	require.True(t, found, "aws2-sqs scaler should be registered")
	assert.Equal(t, "aws2-sqs", mapper.Component())
}

func TestAWS2SQSScalerMap(t *testing.T) {
	scaler := &AWS2SQSScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		expectedMeta map[string]string
	}{
		{
			name:      "explicit queue URL",
			pathValue: "orders",
			params: map[string]string{
				"region":   "eu-west-1",
				"queueUrl": "https://sqs.eu-west-1.amazonaws.com/123456789012/orders",
			},
			expectedMeta: map[string]string{
				"awsRegion": "eu-west-1",
				"queueURL":  "https://sqs.eu-west-1.amazonaws.com/123456789012/orders",
			},
		},
		{
			name:      "queue name with owner account and region",
			pathValue: "orders",
			params:    map[string]string{"region": "us-east-1", "queueOwnerAWSAccountId": "123456789012"},
			expectedMeta: map[string]string{
				"awsRegion": "us-east-1",
				"queueURL":  "https://sqs.us-east-1.amazonaws.com/123456789012/orders",
			},
		},
		{
			name:      "queue ARN",
			pathValue: "arn:aws:sqs:ap-south-1:123456789012:orders.fifo",
			params:    map[string]string{},
			expectedMeta: map[string]string{
				"awsRegion": "ap-south-1",
				"queueURL":  "https://sqs.ap-south-1.amazonaws.com/123456789012/orders.fifo",
			},
		},
		{
			name:      "double-slash URI form",
			pathValue: "//orders",
			params:    map[string]string{"region": "us-east-1", "queueOwnerAWSAccountId": "123456789012"},
			expectedMeta: map[string]string{
				"awsRegion": "us-east-1",
				"queueURL":  "https://sqs.us-east-1.amazonaws.com/123456789012/orders",
			},
		},
		{
			name:         "queue name and region only",
			pathValue:    "orders",
			params:       map[string]string{"region": "us-east-1"},
			expectedMeta: map[string]string{"awsRegion": "us-east-1", "queueURL": "orders"},
		},
		{
			name:      "endpoint override",
			pathValue: "orders",
			params: map[string]string{
				"region":                 "us-east-1",
				"queueOwnerAWSAccountId": "000000000000",
				"overrideEndpoint":       "true",
				"uriEndpointOverride":    "http://localstack:4566",
			},
			expectedMeta: map[string]string{
				"awsRegion":   "us-east-1",
				"queueURL":    "https://sqs.us-east-1.amazonaws.com/000000000000/orders",
				"awsEndpoint": "http://localstack:4566",
			},
		},
		{
			name:      "endpoint override not enabled",
			pathValue: "orders",
			params: map[string]string{
				"region":              "us-east-1",
				"uriEndpointOverride": "http://localstack:4566",
			},
			expectedMeta: map[string]string{"awsRegion": "us-east-1", "queueURL": "orders"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.Map(tt.pathValue, tt.params)
			assert.Equal(t, "aws-sqs-queue", kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerAWS2SQS(t *testing.T) {
	tests := []struct {
		name         string
		uri          string
		expectedMeta map[string]string
	}{
		{
			name: "queue name with static credentials",
			uri:  "aws2-sqs:my-queue?region=us-east-1&accessKey=test&secretKey=test",
			expectedMeta: map[string]string{
				"awsRegion": "us-east-1",
				"queueURL":  "my-queue",
			},
		},
		{
			name: "endpoint override",
			uri:  "aws2-sqs:my-queue?region=us-east-1&accessKey=test&secretKey=test&overrideEndpoint=true&uriEndpointOverride=http://elasticmq:9324",
			expectedMeta: map[string]string{
				"awsRegion":   "us-east-1",
				"queueURL":    "my-queue",
				"awsEndpoint": "http://elasticmq:9324",
			},
		},
		{
			name: "queue owner account",
			uri:  "aws2-sqs:my-queue?region=eu-west-1&queueOwnerAWSAccountId=123456789012",
			expectedMeta: map[string]string{
				"awsRegion": "eu-west-1",
				"queueURL":  "https://sqs.eu-west-1.amazonaws.com/123456789012/my-queue",
			},
		},
		{
			name: "queue ARN",
			uri:  "aws2-sqs:arn:aws:sqs:us-east-1:123456789012:my-queue?accessKey=test&secretKey=test",
			expectedMeta: map[string]string{
				"awsRegion": "us-east-1",
				"queueURL":  "https://sqs.us-east-1.amazonaws.com/123456789012/my-queue",
			},
		},
		{
			name: "double-slash form",
			uri:  "aws2-sqs://my-queue?region=us-east-1",
			expectedMeta: map[string]string{
				"awsRegion": "us-east-1",
				"queueURL":  "my-queue",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			trigger, err := keda.MapToKedaTrigger(tt.uri)
			require.NoError(t, err)
			require.NotNil(t, trigger)
			assert.Equal(t, "aws-sqs-queue", trigger.Type)
			assert.Equal(t, tt.expectedMeta, trigger.Metadata)
		})
	}
}
