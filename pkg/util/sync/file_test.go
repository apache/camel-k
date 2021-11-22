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

package sync

import (
	"context"
	"io/ioutil"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {
	file, err := os.CreateTemp("", "camel-k-test-*")
	assert.Nil(t, err)
	defer os.Remove(file.Name())

	ctx, cancel := context.WithDeadline(context.Background(), time.Now().Add(100*time.Second))
	defer cancel()
	changes, err := File(ctx, file.Name())
	assert.Nil(t, err)

	time.Sleep(100 * time.Millisecond)
	expectedNumChanges := 3
	for i := 0; i < expectedNumChanges; i++ {
		if err := ioutil.WriteFile(file.Name(), []byte("data-"+strconv.Itoa(i)), 0o600); err != nil {
			t.Error(err)
		}
		time.Sleep(350 * time.Millisecond)
	}

	numChanges := 0
watch:
	for {
		select {
		case <-ctx.Done():
			return
		case <-changes:
			numChanges++
			if numChanges == expectedNumChanges {
				break watch
			}
		}
	}

	assert.Equal(t, expectedNumChanges, numChanges)
}
