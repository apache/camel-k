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

package advanced

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestOperatorIDFiltering(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsop1 string) {
			operator1 := "operator-1"
			InstallOperatorWithID(t, ctx, g, nsop1, operator1)
			g.Eventually(PlatformPhase(t, ctx, nsop1), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsop2 string) {
				operator2 := "operator-2"
				InstallOperatorWithID(t, ctx, g, nsop2, operator2)
				g.Eventually(PlatformPhase(t, ctx, nsop2), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

				t.Run("Operators ignore non-scoped integrations", func(t *testing.T) {
					g.Expect(KamelRunWithID(t, ctx, "operator-x", ns, "files/yaml.yaml", "--name", "untouched", "--force").Execute()).To(Succeed())
					g.Consistently(IntegrationPhase(t, ctx, ns, "untouched"), 10*time.Second).Should(BeEmpty())
				})

				t.Run("Operators run scoped integrations", func(t *testing.T) {
					g.Expect(KamelRunWithID(t, ctx, "operator-x", ns, "files/yaml.yaml", "--name", "moving", "--force").Execute()).To(Succeed())
					g.Expect(AssignIntegrationToOperator(t, ctx, ns, "moving", operator1)).To(Succeed())
					g.Eventually(IntegrationPhase(t, ctx, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ctx, ns, "moving"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ctx, ns, "moving"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				})

				t.Run("Operators can handoff scoped integrations", func(t *testing.T) {
					g.Expect(AssignIntegrationToOperator(t, ctx, ns, "moving", operator2)).To(Succeed())
					g.Eventually(IntegrationPhase(t, ctx, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseBuildingKit))
					g.Eventually(IntegrationPhase(t, ctx, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ctx, ns, "moving"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ctx, ns, "moving"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				})

				t.Run("Operators can be deactivated after completely handing off scoped integrations", func(t *testing.T) {
					g.Expect(ScaleOperator(t, ctx, nsop1, 0)).To(Succeed())
					g.Expect(Kamel(t, ctx, "rebuild", "-n", ns, "moving").Execute()).To(Succeed())
					g.Eventually(IntegrationPhase(t, ctx, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ctx, ns, "moving"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ctx, ns, "moving"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
					g.Expect(ScaleOperator(t, ctx, nsop1, 1)).To(Succeed())
				})

				t.Run("Operators can run scoped integrations with fixed image", func(t *testing.T) {
					kitName := IntegrationKit(t, ctx, ns, "moving")()
					g.Expect(kitName).NotTo(BeEmpty())
					kitImage := KitImage(t, ctx, nsop2, kitName)()
					g.Expect(kitImage).NotTo(BeEmpty())
					// external kit creation
					externalKit := v1.IntegrationKit{
						ObjectMeta: metav1.ObjectMeta{
							Namespace: ns,
							Name:      "external",
							Labels: map[string]string{
								v1.IntegrationKitTypeLabel: v1.IntegrationKitTypeExternal,
							},
							Annotations: map[string]string{
								"camel.apache.org/operator.id": operator2,
							},
						},
						Spec: v1.IntegrationKitSpec{
							Image: kitImage,
						},
					}
					g.Expect(TestClient(t).Create(ctx, &externalKit)).Should(BeNil())
					g.Expect(KamelRunWithID(t, ctx, operator2, ns, "files/yaml.yaml", "--name", "pre-built", "--kit", "external", "--force").Execute()).To(Succeed())
					g.Consistently(IntegrationPhase(t, ctx, ns, "pre-built"), 10*time.Second).ShouldNot(Equal(v1.IntegrationPhaseBuildingKit))
					g.Eventually(IntegrationPhase(t, ctx, ns, "pre-built"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationStatusImage(t, ctx, ns, "pre-built"), TestTimeoutShort).Should(Equal(kitImage))
					g.Eventually(IntegrationPodPhase(t, ctx, ns, "pre-built"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ctx, ns, "pre-built"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
					g.Expect(Kamel(t, ctx, "delete", "pre-built", "-n", ns).Execute()).To(Succeed())
				})

				t.Run("Operators can run scoped Pipes", func(t *testing.T) {
					g.Expect(KamelBindWithID(t, ctx, "operator-x", ns, "timer-source?message=Hello", "log-sink", "--name", "klb", "--force").Execute()).To(Succeed())
					g.Consistently(Integration(t, ctx, ns, "klb"), 10*time.Second).Should(BeNil())
					g.Expect(AssignPipeToOperator(t, ctx, ns, "klb", operator1)).To(Succeed())
					g.Eventually(Integration(t, ctx, ns, "klb"), TestTimeoutShort).ShouldNot(BeNil())
					g.Eventually(IntegrationPhase(t, ctx, ns, "klb"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ctx, ns, "klb"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				})
			})
		})
	})
}
