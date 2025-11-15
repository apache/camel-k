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
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/internal"
	corev1 "k8s.io/api/core/v1"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1"

	"github.com/stretchr/testify/require"
)

func TestValidateEndpoint(t *testing.T) {
	testcases := []struct {
		name              string
		namespace         string
		operatorNamespace string
		endpoint          v1.Endpoint
	}{
		{
			name:      "kamelet-ref",
			namespace: "test",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1.KameletKind,
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:      "kamelet-ref-in-namespace",
			namespace: "test",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1.KameletKind,
					Namespace:  "test",
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:              "kamelet-ref-in-operator-namespace",
			namespace:         "test",
			operatorNamespace: "global",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1.KameletKind,
					Namespace:  "global",
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:      "knative-broker-ref",
			namespace: "test",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					APIVersion: eventing.SchemeGroupVersion.String(),
					Name:       "foo-broker",
				},
			},
		},
		{
			name:      "knative-broker-ref-in-namespace",
			namespace: "test",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					Namespace:  "test",
					APIVersion: eventing.SchemeGroupVersion.String(),
					Name:       "foo-broker",
				},
			},
		},
		{
			name:      "it-ref",
			namespace: "test",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1.IntegrationKind,
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-it",
				},
			},
		},
		{
			name:      "pipe-ref",
			namespace: "test",
			endpoint: v1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1.PipeKind,
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-pipe",
				},
			},
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := internal.NewFakeClient()
			require.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: tc.namespace,
				Profile:   v1.DefaultTraitProfile,
			}
			if tc.operatorNamespace != "" {
				// special privileges required for cross namespace
				bindingContext.ServiceAccountName = "cross-ns-sa"
				bindingContext.Namespace = "default"
			}

			err = validateEndpoint(bindingContext, tc.endpoint)
			require.NoError(t, err)
		})
	}
}

func TestValidateEndpointErrorRefURI(t *testing.T) {
	uri := "log:info"

	endpoint := v1.Endpoint{
		URI: &uri,
		Ref: &corev1.ObjectReference{
			Kind:       v1.KameletKind,
			APIVersion: v1.SchemeGroupVersion.String(),
			Name:       "foo-kamelet",
		},
	}

	bindingContext := BindingContext{
		Namespace: "default",
	}

	err := validateEndpoint(bindingContext, endpoint)
	require.Error(t, err)
	require.Equal(t, "cannot use both ref and URI to specify an endpoint: only one of them should be used", err.Error())
}

func TestValidateEndpointKameletCrossNS(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	endpoint := v1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       v1.KameletKind,
			APIVersion: v1.SchemeGroupVersion.String(),
			Name:       "foo-kamelet",
			Namespace:  "kamelet-ns",
		},
	}

	bindingContext := BindingContext{
		Namespace:          "default",
		Client:             client,
		Ctx:                context.Background(),
		ServiceAccountName: "cross-ns-sa",
	}

	err = validateEndpoint(bindingContext, endpoint)
	require.NoError(t, err)
}

func TestValidateEndpointKameletCrossNSNoSA(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	endpoint := v1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       v1.KameletKind,
			APIVersion: v1.SchemeGroupVersion.String(),
			Name:       "foo-kamelet",
			Namespace:  "kamelet-ns",
		},
	}

	bindingContext := BindingContext{
		Namespace: "default",
		Client:    client,
		Ctx:       context.Background(),
	}

	err = validateEndpoint(bindingContext, endpoint)
	require.Error(t, err)
	require.Equal(t, "you must to use an authorized ServiceAccount to access cross-namespace resources kamelets. "+
		"Set it in the Pipe spec accordingly", err.Error())
}

func TestValidateEndpointKameletCrossNSDenied(t *testing.T) {
	client, err := internal.NewFakeClient()
	require.NoError(t, err)

	endpoint := v1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       v1.KameletKind,
			APIVersion: v1.SchemeGroupVersion.String(),
			Name:       "foo-kamelet",
			Namespace:  "kamelet-ns",
		},
	}

	bindingContext := BindingContext{
		Namespace:          "default",
		Client:             client,
		Ctx:                context.Background(),
		ServiceAccountName: "my-sa",
	}

	err = validateEndpoint(bindingContext, endpoint)
	require.Error(t, err)
	require.Equal(t, "cross-namespace Pipe reference authorization denied for the ServiceAccount my-sa"+
		" and resources kamelets", err.Error())
}

func TestValidateEndpointErrorKnativeCrossNS(t *testing.T) {
	endpoint := v1.Endpoint{
		Ref: &corev1.ObjectReference{
			Kind:       "Broker",
			Namespace:  "knative-ns",
			APIVersion: eventing.SchemeGroupVersion.String(),
			Name:       "foo-broker",
		},
	}

	bindingContext := BindingContext{
		Namespace: "default",
	}

	err := validateEndpoint(bindingContext, endpoint)
	require.Error(t, err)
	require.Equal(t, "cross-namespace Pipe references are not allowed for Knative", err.Error())
}
