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

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDigestUsesAnnotations(t *testing.T) {
	it := v1.Integration{}
	digest1, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)

	it.Annotations = map[string]string{
		"another.annotation": "hello",
	}
	digest2, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)
	assert.Equal(t, digest1, digest2)

	it.Annotations = map[string]string{
		"another.annotation":                       "hello",
		v1.TraitAnnotationPrefix + "cron.fallback": "true",
	}
	digest3, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)
	assert.NotEqual(t, digest1, digest3)
}

func TestDigestSHA1FromTempFile(t *testing.T) {
	var tmpFile *os.File
	var err error
	if tmpFile, err = os.CreateTemp("", "camel-k-"); err != nil {
		t.Error(err)
	}

	assert.Nil(t, tmpFile.Close())
	require.NoError(t, os.WriteFile(tmpFile.Name(), []byte("hello test!"), 0o400))

	sha1, err := ComputeSHA1(tmpFile.Name())
	require.NoError(t, err)
	assert.Equal(t, "OXPdxTeLf5rqnsqvTi0CgmWoN/0=", sha1)
}

func TestDigestUsesConfigmap(t *testing.T) {
	it := v1.Integration{
		Spec: v1.IntegrationSpec{
			Traits: v1.Traits{
				Mount: &trait.MountTrait{
					Configs:   []string{"configmap:cm"},
					HotReload: ptr.To(true),
				},
			},
		},
	}

	digest1, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)
	cms := []string{"123456"}

	digest2, err := ComputeForIntegration(&it, cms, nil)
	require.NoError(t, err)
	assert.NotEqual(t, digest1, digest2)

	cms = []string{"1234567"}
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
					HotReload: ptr.To(true),
				},
			},
		},
	}

	digest1, err := ComputeForIntegration(&it, nil, nil)
	require.NoError(t, err)
	secrets := []string{"123456"}
	digest2, err := ComputeForIntegration(&it, nil, secrets)
	require.NoError(t, err)
	assert.NotEqual(t, digest1, digest2)

	secrets = []string{"1234567"}
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
	require.NoError(t, err)
	itSpecOnlyTraitUpdatedDigest, err := ComputeForIntegration(&itSpecOnlyTraitUpdated, nil, nil)
	require.NoError(t, err)

	assert.NotEqual(t, itSpecOnlyTraitUpdatedDigest, itDigest, "Digests must not be equal")
}
