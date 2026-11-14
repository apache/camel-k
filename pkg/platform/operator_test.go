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

package platform

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetIntegrationMaxConcurrentReconciles(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		assert.Equal(t, 1, GetIntegrationMaxConcurrentReconciles())
	})

	t.Run("custom valid value", func(t *testing.T) {
		t.Setenv(OperatorIntegrationMaxConcurrentReconcilesEnvVar, "5")
		assert.Equal(t, 5, GetIntegrationMaxConcurrentReconciles())
	})

	t.Run("invalid non-numeric value", func(t *testing.T) {
		t.Setenv(OperatorIntegrationMaxConcurrentReconcilesEnvVar, "invalid")
		assert.Equal(t, 1, GetIntegrationMaxConcurrentReconciles())
	})

	t.Run("invalid non-positive value", func(t *testing.T) {
		t.Setenv(OperatorIntegrationMaxConcurrentReconcilesEnvVar, "0")
		assert.Equal(t, 1, GetIntegrationMaxConcurrentReconciles())

		t.Setenv(OperatorIntegrationMaxConcurrentReconcilesEnvVar, "-5")
		assert.Equal(t, 1, GetIntegrationMaxConcurrentReconciles())
	})
}

func TestGetPipeMaxConcurrentReconciles(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		assert.Equal(t, 1, GetPipeMaxConcurrentReconciles())
	})

	t.Run("custom valid value", func(t *testing.T) {
		t.Setenv(OperatorPipeMaxConcurrentReconcilesEnvVar, "4")
		assert.Equal(t, 4, GetPipeMaxConcurrentReconciles())
	})

	t.Run("invalid non-numeric value", func(t *testing.T) {
		t.Setenv(OperatorPipeMaxConcurrentReconcilesEnvVar, "abc")
		assert.Equal(t, 1, GetPipeMaxConcurrentReconciles())
	})

	t.Run("invalid non-positive value", func(t *testing.T) {
		t.Setenv(OperatorPipeMaxConcurrentReconcilesEnvVar, "0")
		assert.Equal(t, 1, GetPipeMaxConcurrentReconciles())

		t.Setenv(OperatorPipeMaxConcurrentReconcilesEnvVar, "-2")
		assert.Equal(t, 1, GetPipeMaxConcurrentReconciles())
	})
}

func TestGetClientQPS(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		assert.Equal(t, float32(20), GetClientQPS())
	})

	t.Run("custom valid value", func(t *testing.T) {
		t.Setenv(OperatorClientQPSEnvVar, "50")
		assert.Equal(t, float32(50), GetClientQPS())
	})

	t.Run("invalid non-numeric value", func(t *testing.T) {
		t.Setenv(OperatorClientQPSEnvVar, "not-a-number")
		assert.Equal(t, float32(20), GetClientQPS())
	})

	t.Run("invalid non-positive value", func(t *testing.T) {
		t.Setenv(OperatorClientQPSEnvVar, "0")
		assert.Equal(t, float32(20), GetClientQPS())

		t.Setenv(OperatorClientQPSEnvVar, "-10")
		assert.Equal(t, float32(20), GetClientQPS())
	})
}

func TestGetClientBurst(t *testing.T) {
	t.Run("default value", func(t *testing.T) {
		assert.Equal(t, 200, GetClientBurst())
	})

	t.Run("custom valid value", func(t *testing.T) {
		t.Setenv(OperatorClientBurstEnvVar, "350")
		assert.Equal(t, 350, GetClientBurst())
	})

	t.Run("invalid non-numeric value", func(t *testing.T) {
		t.Setenv(OperatorClientBurstEnvVar, "bad")
		assert.Equal(t, 200, GetClientBurst())
	})

	t.Run("invalid non-positive value", func(t *testing.T) {
		t.Setenv(OperatorClientBurstEnvVar, "0")
		assert.Equal(t, 200, GetClientBurst())

		t.Setenv(OperatorClientBurstEnvVar, "-100")
		assert.Equal(t, 200, GetClientBurst())
	})
}
