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

func TestIBMMQScalerRegistered(t *testing.T) {
	mapper, found := keda.GetMapper("jms")
	require.True(t, found, "IBM MQ mapper should be registered for jms")
	assert.Equal(t, "jms", mapper.Component())
}

func TestIBMMQScalerMapWithAutoMetadata(t *testing.T) {
	scaler := &IBMMQScaler{}
	validMetadata := map[string]map[string]string{
		"ibmmq": {
			"queueType": "ibm",
			"host":      "https://mq.example.com/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc",
		},
	}

	tests := []struct {
		name         string
		pathValue    string
		params       map[string]string
		autoMetadata map[string]map[string]string
		expectedType string
		expectedMeta map[string]string
	}{
		{
			name:         "queue with prefix",
			pathValue:    "queue:ORDERS",
			autoMetadata: validMetadata,
			expectedType: "ibmmq",
			expectedMeta: map[string]string{"queueName": "ORDERS"},
		},
		{
			name:         "queue without prefix defaults to queue",
			pathValue:    "ORDERS",
			autoMetadata: validMetadata,
			expectedType: "ibmmq",
			expectedMeta: map[string]string{"queueName": "ORDERS"},
		},
		{
			name:         "topic is not mapped",
			pathValue:    "topic:ORDERS",
			autoMetadata: validMetadata,
		},
		{
			name:         "temporary queue is not mapped",
			pathValue:    "temp-queue:ORDERS",
			autoMetadata: validMetadata,
		},
		{
			name:         "topic destination type is not mapped",
			pathValue:    "ORDERS",
			params:       map[string]string{"destinationType": "topic"},
			autoMetadata: validMetadata,
		},
		{
			name:         "missing queue type discriminator",
			pathValue:    "queue:ORDERS",
			autoMetadata: map[string]map[string]string{
				"ibmmq": {"host": "https://mq.example.com/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc"},
			},
		},
		{
			name:         "non IBM JMS provider is not mapped",
			pathValue:    "queue:ORDERS",
			autoMetadata: map[string]map[string]string{
				"ibmmq": {"queueType": "activemq", "host": "https://mq.example.com/mqsc"},
			},
		},
		{
			name:         "missing admin REST host is not mapped",
			pathValue:    "queue:ORDERS",
			autoMetadata: map[string]map[string]string{
				"ibmmq": {"queueType": "ibm"},
			},
		},
		{
			name:         "empty queue name is not mapped",
			pathValue:    "queue:",
			autoMetadata: validMetadata,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			kedaType, metadata := scaler.MapWithAutoMetadata(tt.pathValue, tt.params, nil, tt.autoMetadata)
			assert.Equal(t, tt.expectedType, kedaType)
			assert.Equal(t, tt.expectedMeta, metadata)
		})
	}
}

func TestMapToKedaTriggerIBMMQ(t *testing.T) {
	autoMetadata := map[string]map[string]string{
		"ibmmq": {
			"queueType": "ibm",
			"host":      "https://mq.example.com/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc",
		},
	}

	trigger, err := keda.MapToKedaTriggerWithAutoMetadata("jms:queue:ORDERS", nil, autoMetadata)
	require.NoError(t, err)
	require.NotNil(t, trigger)
	assert.Equal(t, "ibmmq", trigger.Type)
	assert.Equal(t, map[string]string{"queueName": "ORDERS"}, trigger.Metadata)
}

func TestMapToKedaTriggerIBMMQRequiresExplicitProvider(t *testing.T) {
	trigger, err := keda.MapToKedaTrigger("jms:queue:ORDERS", nil)
	require.NoError(t, err)
	assert.Nil(t, trigger)
}
