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

package platform

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
)

func TestFindIntegrationProfile_UserAnnotationTakesPrecedence(t *testing.T) {
	userProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "user-profile",
			Namespace: "ns",
		},
	}
	nsProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns-profile",
			Namespace: "ns",
		},
	}
	defaultProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ns",
		},
	}
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "ns-profile",
			},
		},
	}

	c, err := internal.NewFakeClient(&userProfile, &nsProfile, &defaultProfile, &namespace)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "user-profile",
			},
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "user-profile", found.Name)
}

func TestFindIntegrationProfile_NamespaceAnnotationTakesPrecedenceOverDefault(t *testing.T) {
	nsProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "ns-profile",
			Namespace: "ns",
		},
	}
	defaultProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ns",
		},
	}
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "ns-profile",
			},
		},
	}

	c, err := internal.NewFakeClient(&nsProfile, &defaultProfile, &namespace)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "ns-profile", found.Name)
}

func TestFindIntegrationProfile_DefaultProfileUsedWhenNoAnnotations(t *testing.T) {
	defaultProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ns",
		},
	}
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}

	c, err := internal.NewFakeClient(&defaultProfile, &namespace)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "default", found.Name)
}

func TestFindIntegrationProfile_OperatorDefaultWhenNoProfile(t *testing.T) {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}

	c, err := internal.NewFakeClient(&namespace)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	assert.Nil(t, found)
}

func TestFindIntegrationProfile_MissingUserAnnotatedProfileReturnsError(t *testing.T) {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
		},
	}

	c, err := internal.NewFakeClient(&namespace)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "missing-user-profile",
			},
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
	assert.Nil(t, found)
}

func TestFindIntegrationProfile_MissingNamespaceAnnotatedProfileReturnsError(t *testing.T) {
	namespace := corev1.Namespace{
		ObjectMeta: metav1.ObjectMeta{
			Name: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "missing-ns-profile",
			},
		},
	}

	c, err := internal.NewFakeClient(&namespace)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
	}

	found, err := findIntegrationProfile(context.TODO(), c, &integration)
	require.Error(t, err)
	assert.True(t, k8serrors.IsNotFound(err))
	assert.Nil(t, found)
}

func TestApplyIntegrationProfile(t *testing.T) {
	defaultProfile := v1.IntegrationProfile{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "default",
			Namespace: "ns",
		},
	}

	c, err := internal.NewFakeClient(&defaultProfile)
	require.NoError(t, err)

	integration := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
		},
	}

	found, err := ApplyIntegrationProfile(context.TODO(), c, &integration)
	require.NoError(t, err)
	require.NotNil(t, found)
	assert.Equal(t, "default", found.Name)
}
