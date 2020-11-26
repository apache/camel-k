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
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned/fake"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestKubernetesEmptyRepository(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset()
	repo := newKubernetesKameletRepository(fakeClient, "test")
	list, err := repo.List(ctx)
	assert.NoError(t, err)
	assert.Len(t, list, 0)
}

func TestKubernetesRepository(t *testing.T) {
	ctx := context.Background()
	fakeClient := fake.NewSimpleClientset(
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
		})
	repo := newKubernetesKameletRepository(fakeClient, "test")
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
