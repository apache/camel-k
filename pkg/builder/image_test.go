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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/camel"
	"github.com/apache/camel-k/pkg/util/cancellable"
	"github.com/apache/camel-k/pkg/util/test"
)

func TestListPublishedImages(t *testing.T) {
	catalog, err := camel.DefaultCatalog()
	assert.Nil(t, err)

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
					"camel.apache.org/kit.type":         v1.IntegrationKitTypePlatform,
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
					"camel.apache.org/kit.type":         v1.IntegrationKitTypePlatform,
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

	assert.Nil(t, err)
	assert.NotNil(t, c)

	i, err := listPublishedImages(&builderContext{
		Client:  c,
		Catalog: catalog,
		C:       cancellable.NewContext(),
	})

	assert.Nil(t, err)
	assert.Len(t, i, 1)
	assert.Equal(t, "image-2", i[0].Image)
}
