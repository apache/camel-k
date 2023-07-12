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

package commonwithcustominstall

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestRunBuildOrderStrategyMatchingDependencies(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-build-order-deps"
		Expect(KamelInstallWithID(operatorID, ns, "--build-order-strategy", string(v1.BuildOrderStrategyDependencies)).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		Expect(CreateTimerKamelet(ns, "timer-source")()).To(Succeed())

		integrationA := "java-a"
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", integrationA,
		).Execute()).To(Succeed())

		Eventually(IntegrationKit(ns, integrationA), TestTimeoutMedium).ShouldNot(BeEmpty())
		integrationKitNameA := IntegrationKit(ns, integrationA)()
		Eventually(Build(ns, integrationKitNameA), TestTimeoutMedium).ShouldNot(BeNil())

		integrationB := "java-b"
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", integrationB,
			"-d", "camel:joor",
		).Execute()).To(Succeed())

		integrationC := "java-c"
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", integrationC,
			"-d", "camel:joor",
			"-d", "camel:zipfile",
		).Execute()).To(Succeed())

		integrationZ := "groovy-z"
		Expect(KamelRunWithID(operatorID, ns, "files/timer-source.groovy",
			"--name", integrationZ,
		).Execute()).To(Succeed())

		Eventually(IntegrationKit(ns, integrationB), TestTimeoutMedium).ShouldNot(BeEmpty())
		Eventually(IntegrationKit(ns, integrationC), TestTimeoutMedium).ShouldNot(BeEmpty())
		Eventually(IntegrationKit(ns, integrationZ), TestTimeoutMedium).ShouldNot(BeEmpty())

		integrationKitNameB := IntegrationKit(ns, integrationB)()
		integrationKitNameC := IntegrationKit(ns, integrationC)()
		integrationKitNameZ := IntegrationKit(ns, integrationZ)()

		Eventually(BuildPhase(ns, integrationKitNameA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationA), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationA, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationA), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameA)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		Eventually(BuildPhase(ns, integrationKitNameB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationB), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationB, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationB), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameB)().Status.BaseImage).Should(ContainSubstring(integrationKitNameA))

		Eventually(BuildPhase(ns, integrationKitNameC), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationC), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationC, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationC), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameC)().Status.BaseImage).Should(ContainSubstring(integrationKitNameB))

		Eventually(BuildPhase(ns, integrationKitNameZ), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationZ), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationZ, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationZ), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameZ)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		buildA := Build(ns, integrationKitNameA)()
		buildB := Build(ns, integrationKitNameB)()
		buildC := Build(ns, integrationKitNameC)()
		buildZ := Build(ns, integrationKitNameZ)()

		Expect(buildA.Status.StartedAt.Before(buildB.Status.StartedAt)).Should(BeTrue())
		Expect(buildA.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
		Expect(buildB.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
		Expect(buildZ.Status.StartedAt.Before(buildB.Status.StartedAt)).Should(BeTrue())
		Expect(buildZ.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunBuildOrderStrategyFIFO(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-build-order-fifo"
		Expect(KamelInstallWithID(operatorID, ns, "--build-order-strategy", string(v1.BuildOrderStrategyFIFO)).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		Expect(CreateTimerKamelet(ns, "timer-source")()).To(Succeed())

		integrationA := "java-a"
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", integrationA,
		).Execute()).To(Succeed())
		Eventually(IntegrationPhase(ns, integrationA)).Should(Equal(v1.IntegrationPhaseBuildingKit))

		integrationB := "java-b"
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", integrationB,
			"-d", "camel:joor",
		).Execute()).To(Succeed())

		integrationZ := "groovy-z"
		Expect(KamelRunWithID(operatorID, ns, "files/timer-source.groovy",
			"--name", integrationZ,
		).Execute()).To(Succeed())

		integrationKitNameA := IntegrationKit(ns, integrationA)()
		Eventually(BuildPhase(ns, integrationKitNameA), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		Eventually(IntegrationPhase(ns, integrationB)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitNameB := IntegrationKit(ns, integrationB)()
		Eventually(BuildPhase(ns, integrationKitNameB), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		Eventually(IntegrationPhase(ns, integrationZ)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitNameZ := IntegrationKit(ns, integrationZ)()
		Eventually(BuildPhase(ns, integrationKitNameZ), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		Eventually(BuildPhase(ns, integrationKitNameA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationA), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationA, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationA), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameA)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		Eventually(BuildPhase(ns, integrationKitNameB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationB), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationB, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationB), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameB)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		Eventually(BuildPhase(ns, integrationKitNameZ), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		Eventually(IntegrationPodPhase(ns, integrationZ), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, integrationZ, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, integrationZ), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(Kit(ns, integrationKitNameZ)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
