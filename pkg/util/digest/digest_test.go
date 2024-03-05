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
	"os"
	"testing"

	"github.com/stretchr/testify/require"

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
)

func TestDigestUsesAnnotations(t *testing.T) {
	it := v1.Integration{}
	digest1, err := ComputeForIntegration(&it, nil, nil)
	assert.NoError(t, err)

	it.Annotations = map[string]string{
		"another.annotation": "hello",
	}
	digest2, err := ComputeForIntegration(&it, nil, nil)
	assert.NoError(t, err)
	assert.Equal(t, digest1, digest2)

	it.Annotations = map[string]string{
		"another.annotation":                   "hello",
		"trait.camel.apache.org/cron.fallback": "true",
	}
	digest3, err := ComputeForIntegration(&it, nil, nil)
	assert.NoError(t, err)
	assert.NotEqual(t, digest1, digest3)
}

func TestDigestSHA1FromTempFile(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	assert.Nil(t, os.WriteFile(tmpFile.Name(), []byte("hello test!"), 0o400))

	sha1, err := ComputeSHA1(tmpFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, "OXPdxTeLf5rqnsqvTi0CgmWoN/0=", sha1)
}

func TestDigestUsesConfigmap(t *testing.T) {
	it := v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Mount: &trait.MountTrait{
					Configs:   []string{"configmap:cm"},
					HotReload: pointer.Bool(true),
				},
			},
		},
	}

	digest1, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)

	cm := corev1.ConfigMap{
		Data: map[string]string{
			"foo": "bar",
		},
	}
	cms := []*corev1.ConfigMap{&cm}

	digest2, err := ComputeForIntegration(&it, cms, nil)
	require.NoError(t, err)
	assert.NotEqual(t, digest1, digest2)

	cm.Data["foo"] = "bar updated"
	digest3, err := ComputeForIntegration(&it, cms, nil)
	require.NoError(t, err)
	assert.NotEqual(t, digest2, digest3)

	digest4, err := ComputeForIntegration(&it, cms, nil)
	require.NoError(t, err)
	assert.Equal(t, digest4, digest3)
}

func TestDigestUsesSecret(t *testing.T) {
	it := v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Mount: &trait.MountTrait{
					Configs:   []string{"secret:mysec"},
					HotReload: pointer.Bool(true),
				},
			},
		},
	}

	digest1, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)

	sec := corev1.Secret{
		Data: map[string][]byte{
			"foo": []byte("bar"),
		},
		StringData: map[string]string{
			"foo2": "bar2",
		},
	}

	secrets := []*corev1.Secret{&sec}

	digest2, err := ComputeForIntegration(&it, nil, secrets)
	require.NoError(t, err)
	assert.NotEqual(t, digest1, digest2)

	sec.Data["foo"] = []byte("bar updated")
	digest3, err := ComputeForIntegration(&it, nil, secrets)
	require.NoError(t, err)
	assert.NotEqual(t, digest2, digest3)
}

func TestDigestMatchingTraitsUpdated(t *testing.T) {
	it := v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Camel: &trait.CamelTrait{
					Properties: []string{"hello=world"},
				},
			},
		},
	}

	itSpecOnlyTraitUpdated := v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Camel: &trait.CamelTrait{
					Properties: []string{"hello=world2"},
				},
			},
		},
	}

	itDigest, err := ComputeForIntegration(&it, nil, nil)
	assert.Nil(t, err)
	itSpecOnlyTraitUpdatedDigest, err := ComputeForIntegration(&itSpecOnlyTraitUpdated, nil, nil)
	assert.Nil(t, err)

	assert.NotEqual(t, itSpecOnlyTraitUpdatedDigest, itDigest, "Digests must not be equal")
}
