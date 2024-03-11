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
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

func TestPipeScale(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-pipe-scale"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithIDAndKameletCatalog(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		ocp, err := openshift.IsOpenShift(TestClient(t))
		require.NoError(t, err)
		if ocp {
			t.Skip("TODO: Temporarily disabled as this test is flaky on OpenShift 3")
			return
		}

		name := RandomizedSuffixName("timer2log")
		g.Expect(KamelBindWithID(t, ctx, operatorID, ns, "timer-source?message=HelloPipe", "log-sink", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("HelloPipe"))

		t.Run("Update Pipe scale spec", func(t *testing.T) {
			g.Expect(ScalePipe(t, ctx, ns, name, 3)).To(Succeed())
			// Check the scale cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Check it also cascades into the Pipe scale subresource Status field
			g.Eventually(PipeStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Check the readiness condition becomes truthy back
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// Finally check the readiness condition becomes truthy back onPipe
			g.Eventually(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		})

		t.Run("ScalePipe with polymorphic client", func(t *testing.T) {
			scaleClient, err := TestClient(t).ScalesClient()
			g.Expect(err).To(BeNil())

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(ctx, v1.SchemeGroupVersion.WithResource("Pipes"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			g.Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			g.Expect(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			g.Eventually(IntegrationSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Then check it cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(2))
			// Check it cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Finally check it cascades into the Pipe scale subresource Status field
			g.Eventually(PipeStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
		})

		t.Run("ScalePipe with Camel K client", func(t *testing.T) {
			camel, err := versioned.NewForConfig(TestClient(t).GetConfig())
			g.Expect(err).To(BeNil())

			// Getter
			PipeScale, err := camel.CamelV1().Pipes(ns).GetScale(ctx, name, metav1.GetOptions{})
			g.Expect(err).To(BeNil())
			g.Expect(PipeScale.Spec.Replicas).To(BeNumerically("==", 2))
			g.Expect(PipeScale.Status.Replicas).To(BeNumerically("==", 2))

			// Setter
			PipeScale.Spec.Replicas = 1
			_, err = camel.CamelV1().Pipes(ns).UpdateScale(ctx, name, PipeScale, metav1.UpdateOptions{})
			g.Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling inPipe
			g.Expect(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Pipe scale subresource Spec field
			g.Eventually(PipeSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
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

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
