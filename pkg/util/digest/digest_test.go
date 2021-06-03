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

package digest

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
)

func TestDigestUsesAnnotations(t *testing.T) {
	it := v1.Integration{}
	digest1, err := ComputeForIntegration(&it)
	assert.NoError(t, err)

	it.Annotations = map[string]string{
		"another.annotation": "hello",
	}
	digest2, err := ComputeForIntegration(&it)
	assert.NoError(t, err)
	assert.Equal(t, digest1, digest2)

	it.Annotations = map[string]string{
		"another.annotation":                   "hello",
		"trait.camel.apache.org/cron.fallback": "true",
	}
	digest3, err := ComputeForIntegration(&it)
	assert.NoError(t, err)
	assert.NotEqual(t, digest1, digest3)
}
