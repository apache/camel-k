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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	eventing "knative.dev/eventing/pkg/apis/eventing/v1"

	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
)

func TestValidateEndpoint(t *testing.T) {
	testcases := []struct {
		name              string
		namespace         string
		operatorNamespace string
		endpoint          v1alpha1.Endpoint
	}{
		{
			name:      "kamelet-ref",
			namespace: "test",
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1alpha1.KameletKind,
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:      "kamelet-ref-in-namespace",
			namespace: "test",
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1alpha1.KameletKind,
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
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1alpha1.KameletKind,
					Namespace:  "global",
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:      "knative-broker-ref",
			namespace: "test",
			endpoint: v1alpha1.Endpoint{
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
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					Namespace:  "test",
					APIVersion: eventing.SchemeGroupVersion.String(),
					Name:       "foo-broker",
				},
			},
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			if tc.operatorNamespace != "" {
				t.Setenv("NAMESPACE", tc.operatorNamespace)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: tc.namespace,
				Profile:   v1.DefaultTraitProfile,
			}

			err = validateEndpoint(bindingContext, tc.endpoint)
			assert.NoError(t, err)
		})
	}
}

func TestValidateEndpointError(t *testing.T) {
	uri := "log:info"

	testcases := []struct {
		name              string
		namespace         string
		operatorNamespace string
		endpoint          v1alpha1.Endpoint
	}{
		{
			name:      "kamelet-ref-and-uri",
			namespace: "test",
			endpoint: v1alpha1.Endpoint{
				URI: &uri,
				Ref: &corev1.ObjectReference{
					Kind:       v1alpha1.KameletKind,
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:      "kamelet-ref-cross-namespace",
			namespace: "test",
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       v1alpha1.KameletKind,
					Namespace:  "other",
					APIVersion: v1.SchemeGroupVersion.String(),
					Name:       "foo-kamelet",
				},
			},
		},
		{
			name:              "knative-broker-ref-in-operator-namespace",
			namespace:         "test",
			operatorNamespace: "global",
			endpoint: v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Broker",
					Namespace:  "global",
					APIVersion: eventing.SchemeGroupVersion.String(),
					Name:       "foo-broker",
				},
			},
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.name), func(t *testing.T) {
			if tc.operatorNamespace != "" {
				t.Setenv("NAMESPACE", tc.operatorNamespace)
			}

			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			bindingContext := BindingContext{
				Ctx:       ctx,
				Client:    client,
				Namespace: tc.namespace,
				Profile:   v1.DefaultTraitProfile,
			}

			err = validateEndpoint(bindingContext, tc.endpoint)
			assert.Error(t, err, "cross-namespace references are not allowed in Pipe")
		})
	}
}
