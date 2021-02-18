// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "knative"

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

package common

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"k8s.io/client-go/dynamic"
	"k8s.io/client-go/restmapper"
	"k8s.io/client-go/scale"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
)

func TestIntegrationScale(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Expect(Kamel("run", "-n", ns, "files/Java.java", "--name", name).Execute()).Should(BeNil())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		t.Run("Scale integration with polymorphic client", func(t *testing.T) {
			// Polymorphic scale client
			groupResources, err := restmapper.GetAPIGroupResources(TestClient().Discovery())
			assert.Nil(t, err)
			mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
			resolver := scale.NewDiscoveryScaleKindResolver(TestClient().Discovery())
			scaleClient, err := scale.NewForConfig(TestClient().GetConfig(), mapper, dynamic.LegacyAPIPathResolverFunc, resolver)
			assert.Nil(t, err)

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(TestContext, camelv1.SchemeGroupVersion.WithResource("integrations"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			if err != nil {
				t.Fatal(err)
			}

			// Check the Integration scale subresource Spec field
			Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Then check it cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutMedium).Should(HaveLen(2))
			// Finally check it cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
		})

		t.Run("Scale integration with Camel K client", func(t *testing.T) {
			camel, err := versioned.NewForConfig(TestClient().GetConfig())
			if err != nil {
				t.Fatal(err)
			}

			// Getter
			integrationScale, err := camel.CamelV1().Integrations(ns).GetScale(TestContext, name, metav1.GetOptions{})
			Expect(integrationScale).ShouldNot(BeNil())
			Expect(integrationScale.Spec.Replicas).Should(BeNumerically("==", 2))
			Expect(integrationScale.Status.Replicas).Should(BeNumerically("==", 2))

			// Setter
			integrationScale.Spec.Replicas = 1
			integrationScale, err = camel.CamelV1().Integrations(ns).UpdateScale(TestContext, name, integrationScale, metav1.UpdateOptions{})
			if err != nil {
				t.Fatal(err)
			}

			// Check the Integration scale subresource Spec field
			Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Then check it cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutMedium).Should(HaveLen(1))
			// Finally check it cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
		})

		// Cleanup
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}
