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

func TestActiveMQScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("activemq")
	require.True(t, found, "activemq scaler should be registered")
	assert.Equal(t, "activemq", mapper.Component())
}

func TestActiveMQScalerMap(t *testing.T) {
	scaler := &ActiveMQScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:      "full activemq params with queue prefix",
			pathValue: "queue:orders",
			params: map[string]string{
				"managementEndpoint":        "activemq:8161",
				"brokerName":                "my-broker",
				"targetQueueSize":           "50",
				"activationTargetQueueSize": "5",
				"corsHeader":                "*",
				"restAPITemplate":           "http://{endpoint}/api/jolokia",
			},
			expectedType: "activemq",
			expectedMeta: map[string]string{
				"destinationName":           "orders",
				"managementEndpoint":        "activemq:8161",
				"brokerName":                "my-broker",
				"targetQueueSize":           "50",
				"activationTargetQueueSize": "5",
				"corsHeader":                "*",
				"restAPITemplate":           "http://{endpoint}/api/jolokia",
			},
		},
		{
			name:         "only destination name without prefix",
			pathValue:    "my-queue",
			params:       map[string]string{},
			expectedType: "activemq",
			expectedMeta: map[string]string{
				"destinationName": "my-queue",
			},
		},
		{
			name:         "destination with topic prefix",
			pathValue:    "topic:notifications",
			params:       map[string]string{"managementEndpoint": "activemq.default:8161"},
			expectedType: "activemq",
			expectedMeta: map[string]string{
				"destinationName":    "notifications",
				"managementEndpoint": "activemq.default:8161",
			},
		},
		{
			name:      "destinationName overridden by param",
			pathValue: "queue:dummy",
			params: map[string]string{
				"destinationName": "actual-queue",
			},
			expectedType: "activemq",
			expectedMeta: map[string]string{
				"destinationName": "actual-queue",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.Map(tt.pathValue, tt.params)
			assert.Equal(t, tt.expectedType, kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerActiveMQ(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("activemq:queue:orders?managementEndpoint=activemq:8161&brokerName=my-broker")
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "activemq", trigger.Type)
	assert.Equal(t, map[string]string{
		"destinationName":    "orders",
		"managementEndpoint": "activemq:8161",
		"brokerName":         "my-broker",
	}, trigger.Metadata)
}
