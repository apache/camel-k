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

package apis

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
)

func TestConditionAccessors(t *testing.T) {
	var c *Condition
	assert.False(t, c.IsTrue())
	assert.False(t, c.IsFalse())
	assert.Empty(t, c.GetReason())
	assert.Empty(t, c.GetMessage())

	c = &Condition{Type: ConditionReady, Status: corev1.ConditionFalse, Reason: "RevisionFailed", Message: "revision failed"}
	assert.False(t, c.IsTrue())
	assert.True(t, c.IsFalse())
	assert.Equal(t, "RevisionFailed", c.GetReason())
	assert.Equal(t, "revision failed", c.GetMessage())

	c.Status = corev1.ConditionTrue
	assert.True(t, c.IsTrue())
	assert.False(t, c.IsFalse())
}
