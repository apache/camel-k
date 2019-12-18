// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "integration"

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

package e2e

import (
	"testing"

	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/stretchr/testify/assert"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func TestBackwardCompatibility(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {

		data := `
apiVersion: ` + v1.SchemeGroupVersion.String() + `
kind: Integration
metadata:
  name: example
  namespace: ` + ns + `
spec:
  thisDoesNotBelongToSpec: hi
  sources:
  - name: hello.groovy
status:
  thisNeitherBelongs:
    at: all
`

		obj, err := kubernetes.LoadRawResourceFromYaml(data)
		assert.Nil(t, err)
		err = testClient.Create(testContext, obj)
		assert.Nil(t, err)

		integration := v1.NewIntegration(ns, "example")
		key, err := client.ObjectKeyFromObject(&integration)
		assert.Nil(t, err)

		unstr := unstructured.Unstructured{
			Object: map[string]interface{}{
				"kind":       "Integration",
				"apiVersion": v1.SchemeGroupVersion.String(),
			},
		}
		err = testClient.Get(testContext, key, &unstr)
		assert.Nil(t, err)
		spec := unstr.Object["spec"]
		assert.NotNil(t, spec)
		attr := spec.(map[string]interface{})["thisDoesNotBelongToSpec"]
		assert.Equal(t, "hi", attr)

		err = testClient.Get(testContext, key, &integration)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(integration.Spec.Sources))
		assert.Equal(t, "hello.groovy", integration.Spec.Sources[0].Name)
	})
}

func TestV1Alpha1Compatibility(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {

		data := `
apiVersion: camel.apache.org/v1alpha1
kind: Integration
metadata:
  name: example
  namespace: ` + ns + `
spec:
  sources:
  - name: hello.groovy
`

		obj, err := kubernetes.LoadRawResourceFromYaml(data)
		assert.Nil(t, err)
		dynClient, err := dynamic.NewForConfig(testClient.GetConfig())
		assert.Nil(t, err)

		obj, err = dynClient.Resource(schema.GroupVersionResource{
			Group: "camel.apache.org",
			// Using old v1alpha1 version for testing
			Version:  "v1alpha1",
			Resource: "integrations",
		}).Namespace(ns).Create(obj.(*unstructured.Unstructured), metav1.CreateOptions{})
		assert.Nil(t, err)

		integration := v1.NewIntegration(ns, "example")
		key, err := client.ObjectKeyFromObject(&integration)
		assert.Nil(t, err)

		err = testClient.Get(testContext, key, &integration)
		assert.Nil(t, err)
		assert.Equal(t, 1, len(integration.Spec.Sources))
		assert.Equal(t, "hello.groovy", integration.Spec.Sources[0].Name)
	})
}
