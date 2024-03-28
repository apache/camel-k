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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestRunIncrementalBuildRoutine(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-incremental-build-routine"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		name := RandomizedSuffixName("java")
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(t, ctx, ns, name)()
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		t.Run("Reuse previous kit", func(t *testing.T) {
			nameClone := "java-clone"
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameClone).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameClone), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameClone, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameClone), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationCloneKitName := IntegrationKit(t, ctx, ns, nameClone)()
			g.Eventually(integrationCloneKitName).Should(Equal(integrationKitName))
		})

		t.Run("Create incremental kit", func(t *testing.T) {
			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameIncremental, "-d", "camel:zipfile").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(t, ctx, ns, nameIncremental)()
			// the container comes in a format like
			// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
			// we should be saving just to check the substring is contained
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunIncrementalBuildPod(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-incremental-build-pod"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		name := RandomizedSuffixName("java")
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "builder.strategy=pod").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(t, ctx, ns, name)()
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
		g.Eventually(BuilderPodsCount(t, ctx, ns)).Should(Equal(1))

		t.Run("Reuse previous kit", func(t *testing.T) {
			nameClone := RandomizedSuffixName("java-clone")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameClone, "-t", "builder.strategy=pod").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameClone), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameClone, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameClone), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationCloneKitName := IntegrationKit(t, ctx, ns, nameClone)()
			g.Eventually(integrationCloneKitName).Should(Equal(integrationKitName))
			g.Eventually(BuilderPodsCount(t, ctx, ns)).Should(Equal(1))
		})

		t.Run("Create incremental kit", func(t *testing.T) {
			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameIncremental, "-d", "camel:zipfile", "-t", "builder.strategy=pod").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(t, ctx, ns, nameIncremental)()
			// the container comes in a format like
			// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
			// we should be saving just to check the substring is contained
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
			g.Eventually(BuilderPodsCount(t, ctx, ns)).Should(Equal(2))
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunIncrementalBuildOff(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-standard-build"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		name := RandomizedSuffixName("java")
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(t, ctx, ns, name)()
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		t.Run("Don't reuse previous kit", func(t *testing.T) {
			t.Skip("FIXME CAMELK: bug on builder.incremental-image-build linked to https://github.com/apache/camel-k/pull/5077 not in camel-k-crds")

			nameClone := RandomizedSuffixName("java-clone")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameClone, "-t", "builder.incremental-image-build=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameClone), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameClone, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameClone), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationCloneKitName := IntegrationKit(t, ctx, ns, nameClone)()
			g.Eventually(Kit(t, ctx, ns, integrationCloneKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		})

		t.Run("Don't create incremental kit", func(t *testing.T) {
			t.Skip("FIXME CAMELK: bug on builder.incremental-image-build linked to https://github.com/apache/camel-k/pull/5077 not in camel-k-crds")

			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameIncremental, "-d", "camel:zipfile", "-t", "builder.incremental-image-build=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(t, ctx, ns, nameIncremental)()
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		})

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunIncrementalBuildWithDifferentBaseImages(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-incremental-different-base"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		name := RandomizedSuffixName("java")
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(t, ctx, ns, name)()
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		t.Run("Create incremental kit", func(t *testing.T) {
			//

			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", nameIncremental, "-d", "camel:zipfile").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(t, ctx, ns, nameIncremental)()
			// the container comes in a format like
			// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
			// we should be save just to check the substring is contained
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
			g.Eventually(Kit(t, ctx, ns, integrationIncrementalKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
		})

		t.Run("Create new hierarchy kit", func(t *testing.T) {
			t.Skip("FIXME CAMELK: CAMEL-20576 - snake-case")

			// We should spin off a new hierarchy of builds
			newBaseImage := "eclipse-temurin:17.0.8.1_1-jdk-ubi9-minimal"
			name = RandomizedSuffixName("java-new")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-d", "camel:mongodb", "-t", fmt.Sprintf("builder.base-image=%s", newBaseImage)).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationKitName = IntegrationKit(t, ctx, ns, name)()
			g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.BaseImage).Should(Equal(newBaseImage))
			g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.RootImage).Should(Equal(newBaseImage))
		})
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
