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
	"encoding/json"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/equality"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestVolatileTimeJSON(t *testing.T) {
	vt := VolatileTime{Inner: metav1.NewTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))}

	data, err := json.Marshal(vt)
	require.NoError(t, err)
	assert.Equal(t, `"2026-01-02T03:04:05Z"`, string(data))

	var parsed VolatileTime
	require.NoError(t, json.Unmarshal(data, &parsed))
	assert.True(t, vt.Inner.Equal(&parsed.Inner))
}

func TestVolatileTimeSemanticEquality(t *testing.T) {
	t1 := VolatileTime{Inner: metav1.NewTime(time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC))}
	t2 := VolatileTime{Inner: metav1.NewTime(time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC))}
	assert.True(t, equality.Semantic.DeepEqual(t1, t2))

	c1 := Condition{Type: ConditionReady, Status: corev1.ConditionTrue, LastTransitionTime: t1}
	c2 := Condition{Type: ConditionReady, Status: corev1.ConditionTrue, LastTransitionTime: t2}
	assert.True(t, equality.Semantic.DeepEqual(c1, c2))

	c2.Status = corev1.ConditionFalse
	assert.False(t, equality.Semantic.DeepEqual(c1, c2))
}
