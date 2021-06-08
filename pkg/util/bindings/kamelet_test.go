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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/test"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestKameletBindingProvider(t *testing.T) {
	testcases := []struct {
		endpointType v1alpha1.EndpointType
		uri          string
		step         map[string]interface{}
	}{
		{
			endpointType: v1alpha1.EndpointTypeSource,
			uri:          "kamelet:mykamelet/source",
			step:         nil,
		},
		{
			endpointType: v1alpha1.EndpointTypeAction,
			uri:          "",
			step: map[string]interface{}{
				"kamelet": map[string]interface{}{
					"name": "mykamelet/action",
				},
			},
		},
		{
			endpointType: v1alpha1.EndpointTypeSink,
			uri:          "kamelet:mykamelet/sink",
			step:         nil,
		},
	}

	for i, tc := range testcases {
		t.Run(fmt.Sprintf("test-%d-%s", i, tc.uri), func(t *testing.T) {
			ctx, cancel := context.WithCancel(context.Background())
			defer cancel()

			client, err := test.NewFakeClient()
			assert.NoError(t, err)

			endpoint := v1alpha1.Endpoint{
				Ref: &corev1.ObjectReference{
					Kind:       "Kamelet",
					APIVersion: "camel.apache.org/v1any1",
					Name:       "mykamelet",
				},
			}

			binding, err := KameletBindingProvider{}.Translate(
				BindingContext{
					Ctx:       ctx,
					Client:    client,
					Namespace: "test",
					Profile:   camelv1.TraitProfileKubernetes,
				},
				EndpointContext{Type: tc.endpointType},
				endpoint)

			assert.NoError(t, err)
			assert.NotNil(t, binding)
			assert.Equal(t, tc.step, binding.Step)
			assert.Equal(t, tc.uri, binding.URI)
		})
	}

}
