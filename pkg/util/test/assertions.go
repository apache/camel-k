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

package test

import (
	"testing"

	"github.com/apache/camel-k/pkg/util/envvar"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
)

// EnvVarExists --.
func EnvVarExists(t *testing.T, env []corev1.EnvVar, name string) {
	t.Helper()

	ev := envvar.Get(env, name)
	assert.NotNil(t, ev)
}

// EnvVarHasValue --.
func EnvVarHasValue(t *testing.T, env []corev1.EnvVar, name string, val string) {
	t.Helper()

	ev := envvar.Get(env, name)
	assert.NotNil(t, ev)
	assert.Equal(t, val, ev.Value)
}

// HasVolume --.
func HasVolume(t *testing.T, volumes []corev1.Volume, name string) {
	t.Helper()

	assert.Condition(t, func() bool {
		for _, v := range volumes {
			if v.Name == name {
				return true
			}
		}
		return false
	})
}
