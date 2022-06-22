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

package repository

import (
	"context"
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
)

func TestURIParse(t *testing.T) {
	tests := []struct {
		uri        string
		error      bool
		repository KameletRepository
	}{
		{
			uri: "github:apache/camel-kamelets",
			repository: &githubKameletRepository{
				owner: "apache",
				repo:  "camel-kamelets",
			},
		},
		{
			uri: "github:apache/camel-kamelets/catalog",
			repository: &githubKameletRepository{
				owner: "apache",
				repo:  "camel-kamelets",
				path:  "catalog",
			},
		},
		{
			uri: "github:apache/camel-kamelets/catalog@v1.2.3",
			repository: &githubKameletRepository{
				owner: "apache",
				repo:  "camel-kamelets",
				path:  "catalog",
				ref:   "v1.2.3",
			},
		},
		{
			uri: "github:apache/camel-kamelets@v1.2.3",
			repository: &githubKameletRepository{
				owner: "apache",
				repo:  "camel-kamelets",
				ref:   "v1.2.3",
			},
		},
		{
			uri:   "github:apache@v1.2.3",
			error: true,
		},
		{
			uri: "github:apache/camel-kamelets/the/path@v1.2.3",
			repository: &githubKameletRepository{
				owner: "apache",
				repo:  "camel-kamelets",
				path:  "the/path",
				ref:   "v1.2.3",
			},
		},
		{
			uri: "github:apache/camel-kamelets/the/path",
			repository: &githubKameletRepository{
				owner: "apache",
				repo:  "camel-kamelets",
				path:  "the/path",
			},
		},
		{
			uri:   "zithub:apache/camel-kamelets/the/path",
			error: true,
		},
		{
			uri:        "none",
			repository: &emptyKameletRepository{},
		},
	}
	for i, test := range tests {
		t.Run(fmt.Sprintf("%d-%s", i, test.uri), func(t *testing.T) {
			catalog, err := newFromURI(test.uri)
			if test.error {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				switch r := test.repository.(type) {
				case *githubKameletRepository:
					gc, ok := catalog.(*githubKameletRepository)
					assert.True(t, ok)
					assert.Equal(t, r.owner, gc.owner)
					assert.Equal(t, r.repo, gc.repo)
					assert.Equal(t, r.path, gc.path)
					assert.Equal(t, r.ref, gc.ref)
				case *emptyKameletRepository:
					_, ok := catalog.(*emptyKameletRepository)
					assert.True(t, ok)
				default:
					t.Fatal("missing case")
				}

			}
		})
	}

}

func TestNewRepository(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset(createTestContext("none")...)
	repo, err := New(ctx, fakeClient, "test")
	assert.NoError(t, err)
	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	k1, err := repo.Get(ctx, "kamelet1")
	assert.NoError(t, err)
	assert.Equal(t, "kamelet1", k1.Name)
	k2, err := repo.Get(ctx, "kamelet2")
	assert.NoError(t, err)
	assert.Equal(t, "kamelet2", k2.Name)
}

func TestNewRepositoryWithCamelKamelets(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset(createTestContext("github:apache/camel-kamelets/kamelets")...)
	repo, err := New(ctx, fakeClient, "test")
	assert.NoError(t, err)
	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.True(t, len(list) > 2)
	k1, err := repo.Get(ctx, "kamelet1")
	assert.NoError(t, err)
	assert.Equal(t, "kamelet1", k1.Name)
	k2, err := repo.Get(ctx, "kamelet2")
	assert.NoError(t, err)
	assert.Equal(t, "kamelet2", k2.Name)
}

func TestNewRepositoryWithDefault(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset(createTestContext()...)
	repo, err := New(ctx, fakeClient, "test")
	assert.NoError(t, err)
	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 2)
	k1, err := repo.Get(ctx, "kamelet1")
	assert.NoError(t, err)
	assert.Equal(t, "kamelet1", k1.Name)
	k2, err := repo.Get(ctx, "kamelet2")
	assert.NoError(t, err)
	assert.Equal(t, "kamelet2", k2.Name)
}

func createTestContext(uris ...string) []runtime.Object {
	res := []runtime.Object{
		&v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "kamelet1",
			},
		},
		&v1alpha1.Kamelet{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "kamelet2",
			},
		},
	}
	if len(uris) > 0 {
		repos := make([]v1.IntegrationPlatformKameletRepositorySpec, 0, len(uris))
		for _, uri := range uris {
			repos = append(repos, v1.IntegrationPlatformKameletRepositorySpec{
				URI: uri,
			})
		}
		res = append(res, &v1.IntegrationPlatform{
			ObjectMeta: metav1.ObjectMeta{
				Namespace: "test",
				Name:      "camel-k",
			},
			Status: v1.IntegrationPlatformStatus{
				IntegrationPlatformSpec: v1.IntegrationPlatformSpec{
					Kamelet: v1.IntegrationPlatformKameletSpec{
						Repositories: repos,
					},
				},
				Phase: v1.IntegrationPlatformPhaseReady,
			},
		})
	}
	return res
}
