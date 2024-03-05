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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestOperatorIDCamelCatalogReconciliation(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, operatorID, ns, "--global", "--force")).To(Succeed())
		g.Eventually(PlatformPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		g.Eventually(DefaultCamelCatalogPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
	})
}

func TestOperatorIDFiltering(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		WithNewTestNamespace(t, func(g *WithT, nsop1 string) {
			operator1 := "operator-1"
			g.Expect(CopyCamelCatalog(t, nsop1, operator1)).To(Succeed())
			g.Expect(CopyIntegrationKits(t, nsop1, operator1)).To(Succeed())
			g.Expect(KamelInstallWithIDAndKameletCatalog(t, operator1, nsop1, "--global", "--force")).To(Succeed())
			g.Eventually(PlatformPhase(t, nsop1), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

			WithNewTestNamespace(t, func(g *WithT, nsop2 string) {
				operator2 := "operator-2"
				g.Expect(CopyCamelCatalog(t, nsop2, operator2)).To(Succeed())
				g.Expect(CopyIntegrationKits(t, nsop2, operator2)).To(Succeed())
				g.Expect(KamelInstallWithIDAndKameletCatalog(t, operator2, nsop2, "--global", "--force")).To(Succeed())
				g.Eventually(PlatformPhase(t, nsop2), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

				t.Run("Operators ignore non-scoped integrations", func(t *testing.T) {
					g.Expect(KamelRunWithID(t, "operator-x", ns, "files/yaml.yaml", "--name", "untouched", "--force").Execute()).To(Succeed())
					g.Consistently(IntegrationPhase(t, ns, "untouched"), 10*time.Second).Should(BeEmpty())
				})

				t.Run("Operators run scoped integrations", func(t *testing.T) {
					g.Expect(KamelRunWithID(t, "operator-x", ns, "files/yaml.yaml", "--name", "moving", "--force").Execute()).To(Succeed())
					g.Expect(AssignIntegrationToOperator(t, ns, "moving", operator1)).To(Succeed())
					g.Eventually(IntegrationPhase(t, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ns, "moving"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ns, "moving"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				})

				t.Run("Operators can handoff scoped integrations", func(t *testing.T) {
					g.Expect(AssignIntegrationToOperator(t, ns, "moving", operator2)).To(Succeed())
					g.Eventually(IntegrationPhase(t, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseBuildingKit))
					g.Eventually(IntegrationPhase(t, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ns, "moving"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ns, "moving"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				})

				t.Run("Operators can be deactivated after completely handing off scoped integrations", func(t *testing.T) {
					g.Expect(ScaleOperator(t, nsop1, 0)).To(Succeed())
					g.Expect(Kamel(t, "rebuild", "-n", ns, "moving").Execute()).To(Succeed())
					g.Eventually(IntegrationPhase(t, ns, "moving"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ns, "moving"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ns, "moving"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
					g.Expect(ScaleOperator(t, nsop1, 1)).To(Succeed())
				})

				t.Run("Operators can run scoped integrations with fixed image", func(t *testing.T) {
					image := IntegrationPodImage(t, ns, "moving")()
					g.Expect(image).NotTo(BeEmpty())
					// Save resources by deleting "moving" integration
					g.Expect(Kamel(t, "delete", "moving", "-n", ns).Execute()).To(Succeed())

					g.Expect(KamelRunWithID(t, "operator-x", ns, "files/yaml.yaml", "--name", "pre-built", "--force",
						"-t", fmt.Sprintf("container.image=%s", image), "-t", "jvm.enabled=true").Execute()).To(Succeed())
					g.Consistently(IntegrationPhase(t, ns, "pre-built"), 10*time.Second).Should(BeEmpty())
					g.Expect(AssignIntegrationToOperator(t, ns, "pre-built", operator2)).To(Succeed())
					g.Eventually(IntegrationPhase(t, ns, "pre-built"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ns, "pre-built"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
					g.Eventually(IntegrationLogs(t, ns, "pre-built"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
					g.Expect(Kamel(t, "delete", "pre-built", "-n", ns).Execute()).To(Succeed())
				})

				t.Run("Operators can run scoped Pipes", func(t *testing.T) {
					g.Expect(KamelBindWithID(t, "operator-x", ns, "timer-source?message=Hello", "log-sink",
						"--name", "klb", "--force").Execute()).To(Succeed())
					g.Consistently(Integration(t, ns, "klb"), 10*time.Second).Should(BeNil())

					g.Expect(AssignPipeToOperator(t, ns, "klb", operator1)).To(Succeed())
					g.Eventually(Integration(t, ns, "klb"), TestTimeoutShort).ShouldNot(BeNil())
					g.Eventually(IntegrationPhase(t, ns, "klb"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
					g.Eventually(IntegrationPodPhase(t, ns, "klb"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				})
			})
		})

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
