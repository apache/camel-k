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

package v1

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIntegrationPlatformBuildPublishStrategy_Validate(t *testing.T) {
	tests := []struct {
		name      string
		input     IntegrationPlatformBuildPublishStrategy
		wantError bool
	}{
		{"valid S2I", IntegrationPlatformBuildPublishStrategyS2I, false},
		{"valid Jib", IntegrationPlatformBuildPublishStrategyJib, false},
		{"invalid strategy", IntegrationPlatformBuildPublishStrategy("wrong"), true},
		{"empty strategy", IntegrationPlatformBuildPublishStrategy(""), true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.input.Validate()
			if tt.wantError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}
