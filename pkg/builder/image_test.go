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

package builder

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/camel"
	"github.com/apache/camel-k/v2/pkg/util/cancellable"
	"github.com/apache/camel-k/v2/pkg/util/test"
)

func TestListPublishedImages(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	require.NoError(t, err)

	c, err := test.NewFakeClient(
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-1",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel:          v1.IntegrationKitTypePlatform,
					"camel.apache.org/runtime.version":  catalog.Runtime.Version,
					"camel.apache.org/runtime.provider": string(catalog.Runtime.Provider),
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase:           v1.IntegrationKitPhaseError,
				Image:           "image-1",
				RuntimeVersion:  catalog.Runtime.Version,
				RuntimeProvider: catalog.Runtime.Provider,
			},
		},
		&v1.IntegrationKit{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationKitKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "ns",
				Name:      "my-kit-2",
				Labels: map[string]string{
					v1.IntegrationKitTypeLabel:          v1.IntegrationKitTypePlatform,
					"camel.apache.org/runtime.version":  catalog.Runtime.Version,
					"camel.apache.org/runtime.provider": string(catalog.Runtime.Provider),
				},
			},
			Status: v1.IntegrationKitStatus{
				Phase:           v1.IntegrationKitPhaseReady,
				Image:           "image-2",
				RuntimeVersion:  catalog.Runtime.Version,
				RuntimeProvider: catalog.Runtime.Provider,
			},
		},
	)

	require.NoError(t, err)
	assert.NotNil(t, c)

	i, err := listPublishedImages(&builderContext{
		Client:  c,
		Catalog: catalog,
		C:       cancellable.NewContext(),
	})

	require.NoError(t, err)
	assert.Len(t, i, 1)
	assert.Equal(t, "image-2", i[0].Image)
}

func TestFindBestImageExactMatch(t *testing.T) {
	requiredArtifacts := []v1.Artifact{
		{
			Checksum: "1",
			ID:       "artifact-1",
		},
		{
			Checksum: "2",
			ID:       "artifact-2",
		},
		{
			Checksum: "3",
			ID:       "artifact-3",
		},
	}
	iks := []v1.IntegrationKitStatus{
		{
			// missing dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
			},
		},
		{
			// exact match
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
			},
		},
		{
			// extra dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		},
		{
			// missing and extra dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		}}

	bestImage, commonLibs := findBestImage(iks, requiredArtifacts)
	assert.NotNil(t, bestImage)
	assert.Equal(t, iks[1], bestImage) // exact match

	assert.NotNil(t, commonLibs)
	assert.Equal(t, 3, len(commonLibs))
	assert.True(t, commonLibs["artifact-1"])
	assert.True(t, commonLibs["artifact-2"])
	assert.True(t, commonLibs["artifact-3"])

}

func TestFindBestImageNoExactMatch(t *testing.T) {
	requiredArtifacts := []v1.Artifact{
		{
			Checksum: "1",
			ID:       "artifact-1",
		},
		{
			Checksum: "2",
			ID:       "artifact-2",
		},
		{
			Checksum: "3",
			ID:       "artifact-3",
		},
	}
	iks := []v1.IntegrationKitStatus{
		{
			// missing 2 dependencies
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
			},
		},
		{
			// missing dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
			},
		},
		{
			// extra dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		},
		{
			// missing and extra dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		}}

	bestImage, commonLibs := findBestImage(iks, requiredArtifacts)
	assert.NotNil(t, bestImage)
	assert.Equal(t, iks[1], bestImage) // missing only 1 dependency and no surplus

	assert.NotNil(t, commonLibs)
	assert.Equal(t, 2, len(commonLibs))
	assert.True(t, commonLibs["artifact-1"])
	assert.True(t, commonLibs["artifact-3"])

}

func TestFindBestImageNoExactMatchBadChecksum(t *testing.T) {
	requiredArtifacts := []v1.Artifact{
		{
			Checksum: "1",
			ID:       "artifact-1",
		},
		{
			Checksum: "2",
			ID:       "artifact-2",
		},
		{
			Checksum: "3",
			ID:       "artifact-3",
		},
	}
	iks := []v1.IntegrationKitStatus{
		{
			// missing 2 dependencies
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
			},
		},
		{
			// missing dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "bad-checksum",
					ID:       "artifact-3",
				},
			},
		},
		{
			// extra dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		},
		{
			// missing and extra dependency
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		}}

	bestImage, commonLibs := findBestImage(iks, requiredArtifacts)
	assert.NotNil(t, bestImage)
	assert.Equal(t, iks[0], bestImage) // 2 missing dependencies and no surplus

	assert.NotNil(t, commonLibs)
	assert.Equal(t, 1, len(commonLibs))
	assert.True(t, commonLibs["artifact-1"])

}

func TestFindBestImageAllImagesWithSurplus(t *testing.T) {
	requiredArtifacts := []v1.Artifact{
		{
			Checksum: "1",
			ID:       "artifact-1",
		},
		{
			Checksum: "2",
			ID:       "artifact-2",
		},
		{
			Checksum: "3",
			ID:       "artifact-3",
		},
	}
	iks := []v1.IntegrationKitStatus{
		{
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "5",
					ID:       "artifact-5",
				},
			},
		},
		{
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
			},
		},
		{
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "3",
					ID:       "artifact-3",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
				{
					Checksum: "5",
					ID:       "artifact-5",
				},
			},
		},
		{
			Artifacts: []v1.Artifact{
				{
					Checksum: "1",
					ID:       "artifact-1",
				},
				{
					Checksum: "2",
					ID:       "artifact-2",
				},
				{
					Checksum: "4",
					ID:       "artifact-4",
				},
				{
					Checksum: "6",
					ID:       "artifact-6",
				},
			},
		}}

	bestImage, commonLibs := findBestImage(iks, requiredArtifacts)
	assert.Equal(t, "", bestImage.Image)
	assert.Empty(t, commonLibs)
}
