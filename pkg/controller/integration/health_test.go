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
	"reflect"
	"testing"

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
	assert.Equal(t, HealthCheckStateDown, health.Status)
	assert.Len(t, health.Checks, 3)
	assert.Equal(t, "camel-routes", health.Checks[0].Name)
	assert.Equal(t, HealthCheckStateDown, health.Checks[0].Status)
	assert.True(t, reflect.DeepEqual(health.Checks[0].Data, map[string]interface{}{
		"route.id":           "route1",
		"route.context.name": "camel-1",
		"route.status":       "Stopped",
	}))
	assert.Equal(t, "context", health.Checks[1].Name)
	assert.Equal(t, HealthCheckStateUp, health.Checks[1].Status)
	assert.True(t, reflect.DeepEqual(health.Checks[1].Data, map[string]interface{}{
		"context.name":    "camel-1",
		"context.version": "3.16.0",
		"context.status":  "Started",
	}))
	assert.Equal(t, "camel-consumers", health.Checks[2].Name)
	assert.Equal(t, HealthCheckStateDown, health.Checks[2].Status)
	assert.True(t, reflect.DeepEqual(health.Checks[2].Data, map[string]interface{}{
		"route.id":           "route1",
		"route.context.name": "camel-1",
		"route.status":       "Stopped",
	}))
}
