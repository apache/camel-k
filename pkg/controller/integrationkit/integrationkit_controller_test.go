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

package integrationkit

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewActions(t *testing.T) {
	reconciler := reconcileIntegrationKit{
		actionFactories: []actionFactory{
			NewInitializeAction,
			NewBuildAction,
			NewMonitorAction,
			NewErrorAction,
		},
	}

	actions := reconciler.newActions()
	require.Len(t, actions, 4)

	assert.IsType(t, &initializeAction{}, actions[0])
	assert.IsType(t, &buildAction{}, actions[1])
	assert.IsType(t, &monitorAction{}, actions[2])
	assert.IsType(t, &errorAction{}, actions[3])

	nextActions := reconciler.newActions()
	assert.NotSame(t, actions[0], nextActions[0])
	assert.NotSame(t, actions[1], nextActions[1])
	assert.NotSame(t, actions[2], nextActions[2])
	assert.NotSame(t, actions[3], nextActions[3])
}
