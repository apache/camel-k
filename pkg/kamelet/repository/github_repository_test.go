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

package repository

import (
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGithubRepository(t *testing.T) {
	if os.Getenv("GITHUB_TOKEN") == "" {
		t.Skip("WARNING: This test requires GITHUB_TOKEN env var")
	}

	ctx := context.Background()
	repo := newGithubKameletRepository("apache", "camel-kamelets", "kamelets", "")
	list, err := repo.List(ctx)
	assert.NoError(t, err)
	require.True(t, len(list) > 0)
	// Repeat multiple times to be sure cache is working and we don't hit rate limits
	maxDistinct := 5
	for i := 0; i < 200; i++ {
		maxPos := maxDistinct
		if len(list) < maxDistinct {
			maxPos = len(list)
		}
		kameletName := list[i%maxPos]
		kamelet, err := repo.Get(ctx, kameletName)
		assert.NoError(t, err)
		assert.Equal(t, kameletName, kamelet.Name)
	}

}
