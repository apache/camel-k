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

	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

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
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		t.Run("Update integration scale spec", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(ScaleIntegration(ns, name, 3)).To(Succeed())
			// Check the readiness condition becomes falsy
			Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionFalse))
			// Check the scale cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Finally check the readiness condition becomes truthy back
			Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
		})

		t.Run("Scale integration with polymorphic client", func(t *testing.T) {
			RegisterTestingT(t)
			// Polymorphic scale client
			groupResources, err := restmapper.GetAPIGroupResources(TestClient().Discovery())
			Expect(err).To(BeNil())
			mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
			resolver := scale.NewDiscoveryScaleKindResolver(TestClient().Discovery())
			scaleClient, err := scale.NewForConfig(TestClient().GetConfig(), mapper, dynamic.LegacyAPIPathResolverFunc, resolver)
			Expect(err).To(BeNil())

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(TestContext, camelv1.SchemeGroupVersion.WithResource("integrations"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			Expect(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady)()).To(Equal(v1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Then check it cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(2))
			// Finally check it cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
		})

		t.Run("Scale integration with Camel K client", func(t *testing.T) {
			RegisterTestingT(t)
			camel, err := versioned.NewForConfig(TestClient().GetConfig())
			Expect(err).To(BeNil())

			// Getter
			integrationScale, err := camel.CamelV1().Integrations(ns).GetScale(TestContext, name, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(integrationScale.Spec.Replicas).To(BeNumerically("==", 2))
			Expect(integrationScale.Status.Replicas).To(BeNumerically("==", 2))

			// Setter
			integrationScale.Spec.Replicas = 1
			integrationScale, err = camel.CamelV1().Integrations(ns).UpdateScale(TestContext, name, integrationScale, metav1.UpdateOptions{})
			Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			Expect(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady)()).To(Equal(v1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Then check it cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(1))
			// Finally check it cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
		})

		// Clean up
		RegisterTestingT(t)
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestKameletBindingScale(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "binding"
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("bind", "timer-source?message=HelloBinding", "log-sink", "-n", ns, "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(KameletBindingCondition(ns, name, camelv1alpha1.KameletBindingConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("HelloBinding"))

		t.Run("Update binding scale spec", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(ScaleKameletBinding(ns, name, 3)).To(Succeed())
			// Check the scale cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Check it also cascades into the KameletBinding scale subresource Status field
			Eventually(KameletBindingStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Check the readiness condition becomes truthy back
			Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
			// Finally check the readiness condition becomes truthy back on kamelet binding
			Eventually(KameletBindingCondition(ns, name, camelv1alpha1.KameletBindingConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
		})

		t.Run("Scale kamelet binding with polymorphic client", func(t *testing.T) {
			RegisterTestingT(t)
			// Polymorphic scale client
			groupResources, err := restmapper.GetAPIGroupResources(TestClient().Discovery())
			Expect(err).To(BeNil())
			mapper := restmapper.NewDiscoveryRESTMapper(groupResources)
			resolver := scale.NewDiscoveryScaleKindResolver(TestClient().Discovery())
			scaleClient, err := scale.NewForConfig(TestClient().GetConfig(), mapper, dynamic.LegacyAPIPathResolverFunc, resolver)
			Expect(err).To(BeNil())

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(TestContext, camelv1alpha1.SchemeGroupVersion.WithResource("kameletbindings"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			Expect(KameletBindingCondition(ns, name, camelv1alpha1.KameletBindingConditionReady)()).To(Equal(v1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Then check it cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(2))
			// Check it cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Finally check it cascades into the KameletBinding scale subresource Status field
			Eventually(KameletBindingStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
		})

		t.Run("Scale kamelet binding with Camel K client", func(t *testing.T) {
			RegisterTestingT(t)
			camel, err := versioned.NewForConfig(TestClient().GetConfig())
			Expect(err).To(BeNil())

			// Getter
			bindingScale, err := camel.CamelV1alpha1().KameletBindings(ns).GetScale(TestContext, name, metav1.GetOptions{})
			Expect(err).To(BeNil())
			Expect(bindingScale.Spec.Replicas).To(BeNumerically("==", 2))
			Expect(bindingScale.Status.Replicas).To(BeNumerically("==", 2))

			// Setter
			bindingScale.Spec.Replicas = 1
			bindingScale, err = camel.CamelV1alpha1().KameletBindings(ns).UpdateScale(TestContext, name, bindingScale, metav1.UpdateOptions{})
			Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling in kamelet binding
			Expect(KameletBindingCondition(ns, name, camelv1alpha1.KameletBindingConditionReady)()).To(Equal(v1.ConditionTrue))
			// Check the KameletBinding scale subresource Spec field
			Eventually(KameletBindingSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Check the readiness condition is still truthy as down-scaling
			Expect(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady)()).To(Equal(v1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Then check it cascades into the Deployment scale
			Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(1))
			// Finally check it cascades into the Integration scale subresource Status field
			Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
		})

		// Clean up
		RegisterTestingT(t)
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
