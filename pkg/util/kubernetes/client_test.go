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

package kubernetes

import (
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestFindIntegrationProfileResource(t *testing.T) {
	integration := v1.Integration{
		TypeMeta: metav1.TypeMeta{
			Kind: v1.IntegrationKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation: "custom",
			},
		},
		Status: v1.IntegrationStatus{
			Phase: v1.IntegrationPhaseRunning,
		},
	}

	profileResource := GetIntegrationProfileResource(&integration)
	assert.Equal(t, "ns/custom", profileResource)

	kit := v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			Kind: v1.IntegrationKitKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation:          "custom",
				v1.IntegrationProfileNamespaceAnnotation: "it-ns",
			},
		},
	}

	profileResource = GetIntegrationProfileResource(&kit)
	assert.Equal(t, "it-ns/custom", profileResource)

	build := v1.Build{
		TypeMeta: metav1.TypeMeta{
			Kind: v1.BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      "test",
			Namespace: "ns",
			Annotations: map[string]string{
				v1.IntegrationProfileAnnotation:          "custom",
				v1.IntegrationProfileNamespaceAnnotation: "it-ns",
			},
		},
	}

	profileResource = GetIntegrationProfileResource(&build)
	assert.Equal(t, "it-ns/custom", profileResource)
}
