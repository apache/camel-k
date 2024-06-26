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

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestRunBuildOrderStrategyMatchingDependencies(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, ctx, g, ns)

		// Update platform with parameters required by this test
		pl := Platform(t, ctx, ns)()
		pl.Spec.Build.MaxRunningBuilds = 4
		pl.Spec.Build.BuildConfiguration.OrderStrategy = v1.BuildOrderStrategyDependencies
		if err := TestClient(t).Update(ctx, pl); err != nil {
			t.Error(err)
			t.FailNow()
		}
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.IntegrationPlatformPhaseReady))

		g.Expect(CreateTimerKamelet(t, ctx, ns, "timer-source")()).To(Succeed())
		integrationA := RandomizedSuffixName("java-a")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", integrationA).Execute()).To(Succeed())

		g.Eventually(IntegrationKit(t, ctx, ns, integrationA), TestTimeoutMedium).ShouldNot(BeEmpty())
		integrationKitNameA := IntegrationKit(t, ctx, ns, integrationA)()
		g.Eventually(Build(t, ctx, ns, integrationKitNameA), TestTimeoutMedium).ShouldNot(BeNil())

		integrationB := RandomizedSuffixName("java-b")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", integrationB, "-d", "camel:cron").Execute()).To(Succeed())

		integrationC := RandomizedSuffixName("java-c")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", integrationC, "-d", "camel:cron", "-d", "camel:zipfile").Execute()).To(Succeed())

		integrationZ := RandomizedSuffixName("yaml-z")
		g.Expect(KamelRun(t, ctx, ns, "files/timer-source.yaml", "--name", integrationZ).Execute()).To(Succeed())

		g.Eventually(IntegrationKit(t, ctx, ns, integrationB), TestTimeoutMedium).ShouldNot(BeEmpty())
		g.Eventually(IntegrationKit(t, ctx, ns, integrationC), TestTimeoutMedium).ShouldNot(BeEmpty())
		g.Eventually(IntegrationKit(t, ctx, ns, integrationZ), TestTimeoutMedium).ShouldNot(BeEmpty())

		integrationKitNameB := IntegrationKit(t, ctx, ns, integrationB)()
		integrationKitNameC := IntegrationKit(t, ctx, ns, integrationC)()
		integrationKitNameZ := IntegrationKit(t, ctx, ns, integrationZ)()

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationA), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationA, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationA), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameA)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationB), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationB, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationB), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameB)().Status.BaseImage).Should(ContainSubstring(integrationKitNameA))

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameC), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationC), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationC, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationC), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameC)().Status.BaseImage).Should(ContainSubstring(integrationKitNameB))

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameZ), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationZ), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationZ, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationZ), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameZ)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		buildA := Build(t, ctx, ns, integrationKitNameA)()
		buildB := Build(t, ctx, ns, integrationKitNameB)()
		buildC := Build(t, ctx, ns, integrationKitNameC)()
		buildZ := Build(t, ctx, ns, integrationKitNameZ)()

		g.Expect(buildA.Status.StartedAt.Before(buildB.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildA.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildB.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildZ.Status.StartedAt.Before(buildB.Status.StartedAt)).Should(BeTrue())
		g.Expect(buildZ.Status.StartedAt.Before(buildC.Status.StartedAt)).Should(BeTrue())
	})
}

func TestRunBuildOrderStrategyFIFO(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, ctx, g, ns)
		// Update platform with parameters required by this test
		pl := Platform(t, ctx, ns)()
		pl.Spec.Build.BuildConfiguration.OrderStrategy = v1.BuildOrderStrategyFIFO
		if err := TestClient(t).Update(ctx, pl); err != nil {
			t.Error(err)
			t.FailNow()
		}
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.IntegrationPlatformPhaseReady))

		g.Expect(CreateTimerKamelet(t, ctx, ns, "timer-source")()).To(Succeed())

		integrationA := RandomizedSuffixName("java-a")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
			"--name", integrationA,
		).Execute()).To(Succeed())
		g.Eventually(IntegrationPhase(t, ctx, ns, integrationA)).Should(Equal(v1.IntegrationPhaseBuildingKit))

		integrationB := RandomizedSuffixName("java-b")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
			"--name", integrationB,
			"-d", "camel:joor",
		).Execute()).To(Succeed())

		integrationZ := RandomizedSuffixName("yaml-z")
		g.Expect(KamelRun(t, ctx, ns, "files/timer-source.yaml",
			"--name", integrationZ,
		).Execute()).To(Succeed())

		integrationKitNameA := IntegrationKit(t, ctx, ns, integrationA)()
		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameA), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		g.Eventually(IntegrationPhase(t, ctx, ns, integrationB)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitNameB := IntegrationKit(t, ctx, ns, integrationB)()
		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameB), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		g.Eventually(IntegrationPhase(t, ctx, ns, integrationZ)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitNameZ := IntegrationKit(t, ctx, ns, integrationZ)()
		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameZ), TestTimeoutShort).Should(Equal(v1.BuildPhaseRunning))

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameA), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationA), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationA, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationA), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameA)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameB), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationB), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationB, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationB), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameB)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		g.Eventually(BuildPhase(t, ctx, ns, integrationKitNameZ), TestTimeoutLong).Should(Equal(v1.BuildPhaseSucceeded))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integrationZ), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationZ, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, integrationZ), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameZ)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
	})
}
