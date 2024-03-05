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

package integration

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/test"

	"github.com/stretchr/testify/assert"
)

func TestGetIntegrationSecretAndConfigmapResourceVersions(t *testing.T) {
	cm := kubernetes.NewConfigMap("default", "cm-test", "test.txt", "test.txt", "xyz", nil)
	sec := &corev1.Secret{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Secret",
			APIVersion: "v1",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "sec-test",
			Namespace: "default",
		},
		Immutable: pointer.Bool(true),
	}
	sec.Data = map[string][]byte{
		"test.txt": []byte("hello"),
	}
	it := &v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "my-it",
			Namespace: "default",
		},
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Mount: &trait.MountTrait{
					Configs:   []string{"configmap:cm-test"},
					Resources: []string{"secret:sec-test"},
				},
			},
		},
	}
	c, err := test.NewFakeClient(cm, sec)
	assert.Nil(t, err)
	// Default hot reload (false)
	configmaps, secrets := getIntegrationSecretAndConfigmapResourceVersions(context.TODO(), c, it)
	assert.Len(t, configmaps, 0)
	assert.Len(t, secrets, 0)
	// Enabled hot reload (true)
	it.Spec.Traits.Mount.HotReload = pointer.Bool(true)
	configmaps, secrets = getIntegrationSecretAndConfigmapResourceVersions(context.TODO(), c, it)
	assert.Len(t, configmaps, 1)
	assert.Len(t, secrets, 1)
	// We cannot guess resource version value. It should be enough to have any non empty value though.
	assert.NotEqual(t, "", configmaps[0])
	assert.NotEqual(t, "", secrets[0])
}
