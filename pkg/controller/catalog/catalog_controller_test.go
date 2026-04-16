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

package catalog

import (
	"testing"

	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestActions(t *testing.T) {
	c, err := internal.NewFakeClient()
	require.NoError(t, err)

	reconciler := reconcileCatalog{
		actions: newCatalogActions(c),
	}

	require.Len(t, reconciler.actions, 2)
	assert.IsType(t, &initializeAction{}, reconciler.actions[0])
	assert.IsType(t, &monitorAction{}, reconciler.actions[1])
	assert.Same(t, c, reconciler.actions[0].(*initializeAction).client)
	assert.Same(t, c, reconciler.actions[1].(*monitorAction).client)
}
