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

package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestValidTolerations(t *testing.T) {
	validTolerations := [][]string{
		{"key=value:NoSchedule"},
		{"key=value:NoExecute"},
		{"key=value:PreferNoSchedule"},
		{"key=value:NoSchedule:120"},
		{"key=value:NoExecute:120"},
		{"key=value:PreferNoSchedule:120"},
		{"existKey:NoSchedule"},
		{"existKey:NoExecute"},
		{"existKey:PreferNoSchedule"},
		{"existKey:NoSchedule:120"},
		{"existKey:NoExecute:120"},
		{"existKey:PreferNoSchedule:120"},
	}
	for _, vd := range validTolerations {
		_, err := NewTolerations(vd)
		assert.Nil(t, err)
	}
}

func TestInvalidTolerations(t *testing.T) {
	validTolerations := [][]string{
		{"key-NoSchedule"},
		{"key=value:Something"},
		{"key@wrong=value:PreferNoSchedule"},
		{"key=value%wrong:NoSchedule:120"},
		{"existKey"},
		{"existKey:"},
		{"existKey:Something"},
		{"existKey:PreferNoSchedule:something"},
	}
	for _, vd := range validTolerations {
		_, err := NewTolerations(vd)
		assert.NotNil(t, err)
	}
}

func TestValueTolerations(t *testing.T) {
	tolerations := []string{
		"key=value:NoSchedule",
		"key=value:NoExecute:120",
		"existKey:PreferNoSchedule",
		"existKey:NoSchedule:120",
	}
	toleration, err := NewTolerations(tolerations)
	assert.Nil(t, err)
	assert.Equal(t, 4, len(toleration))

	assert.Equal(t, "key", toleration[0].Key)
	assert.Equal(t, v1.TolerationOpEqual, toleration[0].Operator)
	assert.Equal(t, "value", toleration[0].Value)
	assert.Equal(t, v1.TaintEffectNoSchedule, toleration[0].Effect)

	assert.Equal(t, "key", toleration[1].Key)
	assert.Equal(t, v1.TolerationOpEqual, toleration[1].Operator)
	assert.Equal(t, "value", toleration[1].Value)
	assert.Equal(t, v1.TaintEffectNoExecute, toleration[1].Effect)
	assert.Equal(t, int64(120), *toleration[1].TolerationSeconds)

	assert.Equal(t, "existKey", toleration[2].Key)
	assert.Equal(t, v1.TolerationOpExists, toleration[2].Operator)
	assert.Equal(t, "", toleration[2].Value)
	assert.Equal(t, v1.TaintEffectPreferNoSchedule, toleration[2].Effect)

	assert.Equal(t, "existKey", toleration[3].Key)
	assert.Equal(t, v1.TolerationOpExists, toleration[3].Operator)
	assert.Equal(t, "", toleration[3].Value)
	assert.Equal(t, v1.TaintEffectNoSchedule, toleration[3].Effect)
	assert.Equal(t, int64(120), *toleration[3].TolerationSeconds)
}

func TestValidNodeSelectors(t *testing.T) {
	validNodeSelectors := [][]string{
		{"key1=value"},
		{"kubernetes.io/hostname=worker0"},
		{"disktype=ssd"},
		{"key=path-to-value"},
		{"keyNum=123"},
	}
	for _, vds := range validNodeSelectors {
		_, err := NewNodeSelectors(vds)
		assert.Nil(t, err)
	}
}

func TestInvalidNodeSelectors(t *testing.T) {
	validNodeSelectors := [][]string{
		{"key1"},
		{"kubernetes.io@hostname=worker0"},
		{"key=path/to/value"},
	}
	for _, vds := range validNodeSelectors {
		_, err := NewNodeSelectors(vds)
		assert.NotNil(t, err)
	}
}

func TestValueNodeSelectors(t *testing.T) {
	nodeSelectorsArray := []string{
		"key=value",
		"kubernetes.io/hostname=worker0",
	}
	nodeSelectors, err := NewNodeSelectors(nodeSelectorsArray)
	assert.Nil(t, err)
	assert.Equal(t, 2, len(nodeSelectors))

	assert.Equal(t, "value", nodeSelectors["key"])
	assert.Equal(t, "worker0", nodeSelectors["kubernetes.io/hostname"])
}
