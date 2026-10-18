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

func TestArtemisScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("amqp")
	require.True(t, found, "artemis scaler should be registered for amqp")
	assert.Equal(t, "amqp", mapper.Component())
}

func TestArtemisScalerMap(t *testing.T) {
	scaler := &ArtemisScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "queue prefix",
			pathValue:    "queue:orders",
			params:       map[string]string{},
			expectedType: "artemis-queue",
			expectedMeta: map[string]string{
				"queueName":     "orders",
				"brokerAddress": "orders",
			},
		},
		{
			name:         "no prefix",
			pathValue:    "orders",
			params:       map[string]string{},
			expectedType: "artemis-queue",
			expectedMeta: map[string]string{
				"queueName":     "orders",
				"brokerAddress": "orders",
			},
		},
		{
			name:         "topic is not mapped",
			pathValue:    "topic:events",
			params:       map[string]string{},
			expectedType: "artemis-queue",
			expectedMeta: map[string]string{},
		},
		{
			name:         "empty path",
			pathValue:    "",
			params:       map[string]string{},
			expectedType: "artemis-queue",
			expectedMeta: map[string]string{},
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

func TestMapToKedaTriggerArtemis(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("amqp:queue:my-queue")
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "artemis-queue", trigger.Type)
	assert.Equal(t, map[string]string{
		"queueName":     "my-queue",
		"brokerAddress": "my-queue",
	}, trigger.Metadata)
}
