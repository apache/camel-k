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
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestRunBuildOrderStrategyMatchingDependencies(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-build-order-deps"
		g.Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorID, ns,
			"--max-running-pipelines", "4",
			"--build-order-strategy", string(v1.BuildOrderStrategyDependencies)).Execute()).To(Succeed())
		g.Eventually(PlatformPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		g.Expect(CreateTimerKamelet(t, operatorID, ns, "timer-source")()).To(Succeed())

		integrationA := RandomizedSuffixName("java-a")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
			"--name", integrationA,
		).Execute()).To(Succeed())

		g.Eventually(IntegrationKit(t, ns, integrationA), TestTimeoutMedium).ShouldNot(BeEmpty())
		integrationKitNameA := IntegrationKit(t, ns, integrationA)()
		g.Eventually(Build(t, ns, integrationKitNameA), TestTimeoutMedium).ShouldNot(BeNil())

		integrationB := RandomizedSuffixName("java-b")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
			"--name", integrationB,
			"-d", "camel:cron",
		).Execute()).To(Succeed())

		integrationC := RandomizedSuffixName("java-c")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
			"--name", integrationC,
			"-d", "camel:cron",
			"-d", "camel:zipfile",
		).Execute()).To(Succeed())

		integrationZ := RandomizedSuffixName("groovy-z")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/timer-source.groovy",
			"--name", integrationZ,
		).Execute()).To(Succeed())

		g.Eventually(IntegrationKit(t, ns, integrationB), TestTimeoutMedium).ShouldNot(BeEmpty())
		g.Eventually(IntegrationKit(t, ns, integrationC), TestTimeoutMedium).ShouldNot(BeEmpty())
		g.Eventually(IntegrationKit(t, ns, integrationZ), TestTimeoutMedium).ShouldNot(BeEmpty())

		integrationKitNameB := IntegrationKit(t, ns, integrationB)()
		integrationKitNameC := IntegrationKit(t, ns, integrationC)()
		integrationKitNameZ := IntegrationKit(t, ns, integrationZ)()

		g.Eventually(BuildPhase(t, ns, integrationKitNameA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationA), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationA, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationA), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameA)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Eventually(BuildPhase(t, ns, integrationKitNameB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationB), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationB, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationB), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameB)().Status.BaseImage).Should(ContainSubstring(integrationKitNameA))

		g.Eventually(BuildPhase(t, ns, integrationKitNameC), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationC), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationC, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationC), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameC)().Status.BaseImage).Should(ContainSubstring(integrationKitNameB))

		g.Eventually(BuildPhase(t, ns, integrationKitNameZ), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationZ), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationZ, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationZ), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameZ)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		buildA := Build(t, ns, integrationKitNameA)()
		buildB := Build(t, ns, integrationKitNameB)()
		buildC := Build(t, ns, integrationKitNameC)()
		buildZ := Build(t, ns, integrationKitNameZ)()

		g.Expect(buildA.Status.StartedAt.Before(buildB.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildA.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildB.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildZ.Status.StartedAt.Before(buildB.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildZ.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

/*
func TestRunBuildOrderStrategyFIFO(t *testing.T) {
	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-build-order-fifo"
		g.Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorID, ns, "--build-order-strategy", string(v1.BuildOrderStrategyFIFO)).Execute()).To(Succeed())
		g.Eventually(PlatformPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		g.Expect(CreateTimerKamelet(t, ns, "timer-source")()).To(Succeed())

		integrationA := RandomizedSuffixName("java-a")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
			"--name", integrationA,
		).Execute()).To(Succeed())
		g.Eventually(IntegrationPhase(t, ns, integrationA)).Should(Equal(v1.IntegrationPhaseBuildingKit))

		integrationB := RandomizedSuffixName("java-b")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
			"--name", integrationB,
			"-d", "camel:joor",
		).Execute()).To(Succeed())

		integrationZ := RandomizedSuffixName("groovy-z")
		g.Expect(KamelRunWithID(t, operatorID, ns, "files/timer-source.groovy",
			"--name", integrationZ,
		).Execute()).To(Succeed())

		integrationKitNameA := IntegrationKit(t, ns, integrationA)()
		g.Eventually(BuildPhase(t, ns, integrationKitNameA), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		g.Eventually(IntegrationPhase(t, ns, integrationB)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitNameB := IntegrationKit(t, ns, integrationB)()
		g.Eventually(BuildPhase(t, ns, integrationKitNameB), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		g.Eventually(IntegrationPhase(t, ns, integrationZ)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitNameZ := IntegrationKit(t, ns, integrationZ)()
		g.Eventually(BuildPhase(t, ns, integrationKitNameZ), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		g.Eventually(BuildPhase(t, ns, integrationKitNameA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationA), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationA, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationA), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameA)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Eventually(BuildPhase(t, ns, integrationKitNameB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationB), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationB, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationB), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameB)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Eventually(BuildPhase(t, ns, integrationKitNameZ), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ns, integrationZ), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, integrationZ, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ns, integrationZ), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ns, integrationKitNameZ)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
*/
