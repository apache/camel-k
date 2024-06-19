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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned"
)

func TestIntegrationScale(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("java")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		t.Run("Update integration scale spec", func(t *testing.T) {
			g.Expect(ScaleIntegration(t, ctx, ns, name, 3)).To(Succeed())
			// Check the readiness condition becomes falsy
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
			// Check the scale cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Finally check the readiness condition becomes truthy back
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		})

		t.Run("Scale integration with polymorphic client", func(t *testing.T) {
			scaleClient, err := TestClient(t).ScalesClient()
			g.Expect(err).To(BeNil())

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(ctx, v1.SchemeGroupVersion.WithResource("integrations"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			g.Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			g.Expect(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			g.Eventually(IntegrationSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Then check it cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(2))
			// Finally check it cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
		})

		t.Run("Scale integration with Camel K client", func(t *testing.T) {
			camel, err := versioned.NewForConfig(TestClient(t).GetConfig())
			g.Expect(err).To(BeNil())

			// Getter
			integrationScale, err := camel.CamelV1().Integrations(ns).GetScale(ctx, name, metav1.GetOptions{})
			g.Expect(err).To(BeNil())
			g.Expect(integrationScale.Spec.Replicas).To(BeNumerically("==", 2))
			g.Expect(integrationScale.Status.Replicas).To(BeNumerically("==", 2))

			// Setter
			integrationScale.Spec.Replicas = 1
			integrationScale, err = camel.CamelV1().Integrations(ns).UpdateScale(ctx, name, integrationScale, metav1.UpdateOptions{})
			g.Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			g.Expect(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			g.Eventually(IntegrationSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Then check it cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(1))
			// Finally check it cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
		})

		t.Run("Scale integration with external image", func(t *testing.T) {
			image := IntegrationPodImage(t, ctx, ns, name)()
			g.Expect(image).NotTo(BeEmpty())
			// Save resources by deleting the integration
			g.Expect(Kamel(t, ctx, "delete", name, "-n", ns).Execute()).To(Succeed())

			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", "pre-built", "-t", fmt.Sprintf("container.image=%s", image)).Execute()).To(Succeed())
			g.Eventually(IntegrationPhase(t, ctx, ns, "pre-built"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "pre-built"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(ScaleIntegration(t, ctx, ns, "pre-built", 0)).To(Succeed())
			g.Eventually(IntegrationPod(t, ctx, ns, "pre-built"), TestTimeoutMedium).Should(BeNil())
			g.Expect(ScaleIntegration(t, ctx, ns, "pre-built", 1)).To(Succeed())
			g.Eventually(IntegrationPhase(t, ctx, ns, "pre-built"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "pre-built"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		})
	})
}
