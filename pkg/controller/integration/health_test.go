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

package integration

import (
	"encoding/json"
	"reflect"
	"testing"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	"github.com/stretchr/testify/assert"
)

func TestNewHealthCheck(t *testing.T) {
	body := []byte(`
		{
			"status": "DOWN",
			"checks": [
			{
				"name": "camel-routes",
				"status": "DOWN",
				"data": {
					"route.id": "route1",
					"route.context.name": "camel-1",
					"route.status": "Stopped"
				}
			},
			{
				"name": "context",
				"status": "UP",
				"data": {
					"context.name": "camel-1",
					"context.version": "3.16.0",
					"context.status": "Started"
				}
			},
			{
				"name": "camel-consumers",
				"status": "DOWN",
				"data": {
					"route.id": "route1",
					"route.context.name": "camel-1",
					"route.status": "Stopped"
				}
			}
			]
		}
	`)
	health, err := NewHealthCheck(body)
	assert.NoError(t, err)
	assert.Equal(t, camelv1.HealthCheckStatusDown, health.Status)
	assert.Len(t, health.Checks, 3)
	assert.Equal(t, "camel-routes", health.Checks[0].Name)
	assert.Equal(t, camelv1.HealthCheckStatusDown, health.Checks[0].Status)
	assert.True(t, reflect.DeepEqual(raw2map(t, health.Checks[0].Data), map[string]interface{}{
		"route.id":           "route1",
		"route.context.name": "camel-1",
		"route.status":       "Stopped",
	}))
	assert.Equal(t, "context", health.Checks[1].Name)
	assert.Equal(t, camelv1.HealthCheckStatusUp, health.Checks[1].Status)
	assert.True(t, reflect.DeepEqual(raw2map(t, health.Checks[1].Data), map[string]interface{}{
		"context.name":    "camel-1",
		"context.version": "3.16.0",
		"context.status":  "Started",
	}))
	assert.Equal(t, "camel-consumers", health.Checks[2].Name)
	assert.Equal(t, camelv1.HealthCheckStatusDown, health.Checks[2].Status)
	assert.True(t, reflect.DeepEqual(raw2map(t, health.Checks[2].Data), map[string]interface{}{
		"route.id":           "route1",
		"route.context.name": "camel-1",
		"route.status":       "Stopped",
	}))
}

func raw2map(t *testing.T, in camelv1.RawMessage) map[string]interface{} {
	t.Helper()
	answer := make(map[string]interface{})

	err := json.Unmarshal(in, &answer)
	assert.Nil(t, err)

	return answer
}
