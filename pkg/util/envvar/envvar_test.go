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

package envvar

import (
	"testing"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
)

func TestGetEnvVar(t *testing.T) {
	vars := []corev1.EnvVar{
		{
			Name:  "MyEnv",
			Value: "MyValue",
		},
	}

	ev := Get(vars, "MyEnv")

	assert.NotNil(t, ev)
	assert.Equal(t, "MyValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)

	assert.Nil(t, Get(vars, "NotMyEnv"))
}

func TestModifyEnvVar(t *testing.T) {
	vars := []corev1.EnvVar{
		{
			Name:  "MyEnv",
			Value: "MyValue",
		},
	}

	ev := Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyValue", ev.Value)

	ev.Value = "MyNewValue"

	ev = Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyNewValue", ev.Value)
}

func TestSetEnvVar(t *testing.T) {
	vars := []corev1.EnvVar{
		{
			Name:  "MyEnv",
			Value: "MyValue",
		},
	}

	ev := Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)

	SetVal(&vars, "MyEnv", "MyNewValue")

	ev = Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyNewValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)

	SetVal(&vars, "MyNewEnv", "MyNewNewValue")

	ev = Get(vars, "MyEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyNewValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)

	ev = Get(vars, "MyNewEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "MyNewNewValue", ev.Value)
	assert.Nil(t, ev.ValueFrom)

	SetValFrom(&vars, "MyNewEnv", "metadata.namespace")

	ev = Get(vars, "MyNewEnv")
	assert.NotNil(t, ev)
	assert.Equal(t, "", ev.Value)
	assert.NotNil(t, ev.ValueFrom)
	assert.Equal(t, "metadata.namespace", ev.ValueFrom.FieldRef.FieldPath)
}
