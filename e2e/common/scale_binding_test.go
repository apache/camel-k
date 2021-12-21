//go:build integration
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

package common

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client/camel/clientset/versioned"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestKameletBindingScale(t *testing.T) {
	ocp, err := openshift.IsOpenShift(TestClient())
	assert.Nil(t, err)
	if ocp {
		t.Skip("TODO: Temporarily disabled as this test is flaky on OpenShift 3")
		return
	}

	WithNewTestNamespace(t, func(ns string) {
		name := "binding"
		Expect(Kamel("install", "-n", ns, "-w").Execute()).To(Succeed())
		Expect(Kamel("bind", "timer-source?message=HelloBinding", "log-sink", "-n", ns, "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(KameletBindingCondition(ns, name, v1alpha1.KameletBindingConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
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
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// Finally check the readiness condition becomes truthy back on kamelet binding
			Eventually(KameletBindingCondition(ns, name, v1alpha1.KameletBindingConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		})

		t.Run("Scale kamelet binding with polymorphic client", func(t *testing.T) {
			RegisterTestingT(t)
			scaleClient, err := TestClient().ScalesClient()
			Expect(err).To(BeNil())

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(TestContext, v1alpha1.SchemeGroupVersion.WithResource("kameletbindings"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			Expect(KameletBindingCondition(ns, name, v1alpha1.KameletBindingConditionReady)()).To(Equal(corev1.ConditionTrue))
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
			_, err = camel.CamelV1alpha1().KameletBindings(ns).UpdateScale(TestContext, name, bindingScale, metav1.UpdateOptions{})
			Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling in kamelet binding
			Expect(KameletBindingCondition(ns, name, v1alpha1.KameletBindingConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the KameletBinding scale subresource Spec field
			Eventually(KameletBindingSpecReplicas(ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Check the readiness condition is still truthy as down-scaling
			Expect(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady)()).To(Equal(corev1.ConditionTrue))
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
