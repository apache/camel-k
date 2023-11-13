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

package misc

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

func TestPipeScale(t *testing.T) {
	RegisterTestingT(t)

	ocp, err := openshift.IsOpenShift(TestClient())
	assert.Nil(t, err)
	if ocp {
		t.Skip("TODO: Temporarily disabled as this test is flaky on OpenShift 3")
		return
	}

	name := RandomizedSuffixName("timer2log")
	Expect(KamelBindWithID(operatorID, ns, "timer-source?message=HelloPipe", "log-sink", "--name", name).Execute()).To(Succeed())
	Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(PipeConditionStatus(ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("HelloPipe"))

	t.Run("Update Pipe scale spec", func(t *testing.T) {
		Expect(ScalePipe(ns, name, 3)).To(Succeed())
		// Check the scale cascades into the Deployment scale
		Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(3))
		// Check it also cascades into the Integration scale subresource Status field
		Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 3)))
		// Check it also cascades into the Pipe scale subresource Status field
		Eventually(PipeStatusReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 3)))
		// Check the readiness condition becomes truthy back
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		// Finally check the readiness condition becomes truthy back onPipe
		Eventually(PipeConditionStatus(ns, name, v1.PipeConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
	})

	t.Run("ScalePipe with polymorphic client", func(t *testing.T) {
		scaleClient, err := TestClient().ScalesClient()
		Expect(err).To(BeNil())

		// Patch the integration scale subresource
		patch := "{\"spec\":{\"replicas\":2}}"
		_, err = scaleClient.Scales(ns).Patch(TestContext, v1.SchemeGroupVersion.WithResource("Pipes"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
		Expect(err).To(BeNil())

		// Check the readiness condition is still truthy as down-scaling
		Expect(PipeConditionStatus(ns, name, v1.PipeConditionReady)()).To(Equal(corev1.ConditionTrue))
		// Check the Integration scale subresource Spec field
		Eventually(IntegrationSpecReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 2)))
		// Then check it cascades into the Deployment scale
		Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(2))
		// Check it cascades into the Integration scale subresource Status field
		Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 2)))
		// Finally check it cascades into the Pipe scale subresource Status field
		Eventually(PipeStatusReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 2)))
	})

	t.Run("ScalePipe with Camel K client", func(t *testing.T) {
		camel, err := versioned.NewForConfig(TestClient().GetConfig())
		Expect(err).To(BeNil())

		// Getter
		PipeScale, err := camel.CamelV1().Pipes(ns).GetScale(TestContext, name, metav1.GetOptions{})
		Expect(err).To(BeNil())
		Expect(PipeScale.Spec.Replicas).To(BeNumerically("==", 2))
		Expect(PipeScale.Status.Replicas).To(BeNumerically("==", 2))

		// Setter
		PipeScale.Spec.Replicas = 1
		_, err = camel.CamelV1().Pipes(ns).UpdateScale(TestContext, name, PipeScale, metav1.UpdateOptions{})
		Expect(err).To(BeNil())

		// Check the readiness condition is still truthy as down-scaling inPipe
		Expect(PipeConditionStatus(ns, name, v1.PipeConditionReady)()).To(Equal(corev1.ConditionTrue))
		// Check the Pipe scale subresource Spec field
		Eventually(PipeSpecReplicas(ns, name), TestTimeoutShort).
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

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
