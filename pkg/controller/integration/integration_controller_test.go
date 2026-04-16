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
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions(t *testing.T) {
	reconciler := reconcileIntegration{
		syntheticActions: append(
			[]Action{
				NewPlatformSetupAction(),
				NewInitializeAction(),
				NewBuildAction(),
				newBuildKitAction(),
				NewBuildCompleteAction(),
			},
			NewMonitorSyntheticAction(),
		),
		nonSyntheticActions: append(
			[]Action{
				NewPlatformSetupAction(),
				NewInitializeAction(),
				NewBuildAction(),
				newBuildKitAction(),
				NewBuildCompleteAction(),
			},
			NewMonitorAction(),
			NewMonitorUnknownAction(),
		),
	}

	t.Run("non-synthetic", func(t *testing.T) {
		actions := reconciler.nonSyntheticActions
		require.Len(t, actions, 7)

		assert.IsType(t, &platformSetupAction{}, actions[0])
		assert.IsType(t, &initializeAction{}, actions[1])
		assert.IsType(t, &buildAction{}, actions[2])
		assert.IsType(t, &buildKitAction{}, actions[3])
		assert.IsType(t, &buildCompleteAction{}, actions[4])
		assert.IsType(t, &monitorAction{}, actions[5])
		assert.IsType(t, &monitorUnknownAction{}, actions[6])
	})

	t.Run("synthetic", func(t *testing.T) {
		actions := reconciler.syntheticActions
		require.Len(t, actions, 6)

		assert.IsType(t, &platformSetupAction{}, actions[0])
		assert.IsType(t, &initializeAction{}, actions[1])
		assert.IsType(t, &buildAction{}, actions[2])
		assert.IsType(t, &buildKitAction{}, actions[3])
		assert.IsType(t, &buildCompleteAction{}, actions[4])
		assert.IsType(t, &monitorSyntheticAction{}, actions[5])
	})
}
