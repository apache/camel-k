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

package bindings

import (
	"context"
	"testing"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestServiceRef(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-svc",
			Namespace: "my-svc-ns",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	client, err := internal.NewFakeClient(&svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-svc-ns",
			Name:       "my-svc",
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
	}

	binding, err := ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, "http://my-svc.my-svc-ns.svc.cluster.local", binding.URI)
}

func TestServiceRefWithOptions(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-svc",
			Namespace: "my-svc-ns",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
			Ports: []corev1.ServicePort{
				{
					Port: 123,
				},
			},
		},
	}

	client, err := internal.NewFakeClient(&svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-svc-ns",
			Name:       "my-svc",
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
		Properties: asEndpointProperties(map[string]string{"path": "/to/my/path"}),
	}

	binding, err := ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, "http://my-svc.my-svc-ns.svc.cluster.local:123/to/my/path", binding.URI)
}

func TestServiceError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-svc-non-cluster-type",
			Namespace: "my-svc-ns",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeExternalName,
		},
	}

	client, err := internal.NewFakeClient(&svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
	}

	// Not a Service
	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-svc-ns",
			Name:       "my-svc",
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "NotAService",
		},
	}

	binding, err := ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.Nil(t, binding)

	// Service Missing
	endpoint = camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-svc-ns",
			Name:       "my-svc-missing",
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
	}

	_, err = ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err)
	assert.Equal(t, "could not load a Service with name my-svc-missing in namespace my-svc-ns", err.Error())

	// Service Not Cluster type
	endpoint = camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-svc-ns",
			Name:       "my-svc-non-cluster-type",
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Service",
		},
	}

	_, err = ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.Error(t, err)
	assert.Equal(t, "operator only supports ClusterIP Service type, feel free to request this support", err.Error())
}

func TestIntegrationRef(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-it",
			Namespace: "my-it-ns",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	client, err := internal.NewFakeClient(&svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-it-ns",
			Name:       "my-it",
			APIVersion: camelv1.SchemeGroupVersion.String(),
			Kind:       "Integration",
		},
	}

	binding, err := ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, "http://my-it.my-it-ns.svc.cluster.local", binding.URI)
}

func TestPipeRef(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	svc := corev1.Service{
		ObjectMeta: v1.ObjectMeta{
			Name:      "my-pipe",
			Namespace: "my-pipe-ns",
		},
		Spec: corev1.ServiceSpec{
			Type: corev1.ServiceTypeClusterIP,
		},
	}

	client, err := internal.NewFakeClient(&svc)
	require.NoError(t, err)

	bindingContext := BindingContext{
		Ctx:       ctx,
		Client:    client,
		Namespace: "test",
	}

	endpoint := camelv1.Endpoint{
		Ref: &corev1.ObjectReference{
			Namespace:  "my-pipe-ns",
			Name:       "my-pipe",
			APIVersion: camelv1.SchemeGroupVersion.String(),
			Kind:       "Pipe",
		},
	}

	binding, err := ServiceRefBindingProvider{}.Translate(bindingContext, EndpointContext{
		Type: camelv1.EndpointTypeSink,
	}, endpoint)
	require.NoError(t, err)
	require.NotNil(t, binding)
	assert.Equal(t, "http://my-pipe.my-pipe-ns.svc.cluster.local", binding.URI)
}
