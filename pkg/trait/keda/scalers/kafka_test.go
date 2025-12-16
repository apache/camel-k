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

func TestKafkaScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("kafka")
	require.True(t, found, "kafka scaler should be registered")
	assert.Equal(t, "kafka", mapper.Component())
}

func TestKafkaScalerMap(t *testing.T) {
	scaler := &KafkaScaler{}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "full kafka params",
			pathValue:    "orders",
			params:       map[string]string{"brokers": "broker:9092", "groupId": "grp"},
			expectedType: "kafka",
			expectedMeta: map[string]string{
				"topic":            "orders",
				"bootstrapServers": "broker:9092",
				"consumerGroup":    "grp",
			},
		},
		{
			name:         "only topic",
			pathValue:    "my-topic",
			params:       map[string]string{},
			expectedType: "kafka",
			expectedMeta: map[string]string{
				"topic": "my-topic",
			},
		},
		{
			name:         "partial params",
			pathValue:    "events",
			params:       map[string]string{"brokers": "localhost:9092"},
			expectedType: "kafka",
			expectedMeta: map[string]string{
				"topic":            "events",
				"bootstrapServers": "localhost:9092",
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

func TestMapToKedaTriggerKafka(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("kafka:orders?brokers=broker:9092&groupId=grp")
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "kafka", trigger.Type)
	assert.Equal(t, map[string]string{
		"topic":            "orders",
		"bootstrapServers": "broker:9092",
		"consumerGroup":    "grp",
	}, trigger.Metadata)
}

func TestMapToKedaTriggerUnsupported(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("timer:tick")
	require.NoError(t, err)
	assert.Nil(t, trigger, "unsupported component should return nil trigger")
}

func TestMapToKedaTriggerEmpty(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("")
	require.NoError(t, err)
	assert.Nil(t, trigger)
}
