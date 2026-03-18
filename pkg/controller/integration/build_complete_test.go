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

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestIntegrationBuildCompleteToDeployDrift(t *testing.T) {
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseBuildComplete,
		},
	}
	c, err := internal.NewFakeClient(it)
	require.NoError(t, err)

	a := buildCompleteAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build-complete", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	require.NotNil(t, handledIt)
	assert.Equal(t, v1.IntegrationPhaseDeploying, handledIt.Status.Phase)
}

func TestIntegrationBuildCompleteDontDeploy(t *testing.T) {
	it := &v1.Integration{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: "ns",
			Name:      "my-it",
			Annotations: map[string]string{
				v1.IntegrationDontRunAfterBuildAnnotation: "true",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseBuildComplete,
		},
	}
	c, err := internal.NewFakeClient(it)
	require.NoError(t, err)

	a := buildCompleteAction{}
	a.InjectLogger(log.Log)
	a.InjectClient(c)
	assert.Equal(t, "build-complete", a.Name())
	assert.True(t, a.CanHandle(it))
	handledIt, err := a.Handle(context.TODO(), it)
	require.NoError(t, err)
	require.NotNil(t, handledIt)
	assert.Equal(t, v1.IntegrationPhaseBuildComplete, handledIt.Status.Phase)
}
