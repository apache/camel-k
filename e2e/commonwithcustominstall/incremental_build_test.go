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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestRunIncrementalBuildRoutine(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-incremental-build"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		name := RandomizedSuffixName("java")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(ns, name)()
		Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		Eventually(Kit(ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		t.Run("Reuse previous kit", func(t *testing.T) {
			nameClone := "java-clone"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameClone,
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameClone), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameClone, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameClone), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationCloneKitName := IntegrationKit(ns, nameClone)()
			Eventually(integrationCloneKitName).Should(Equal(integrationKitName))
		})

		t.Run("Create incremental kit", func(t *testing.T) {
			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameIncremental,
				"-d", "camel:zipfile",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(ns, nameIncremental)()
			// the container comes in a format like
			// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
			// we should be save just to check the substring is contained
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
		})

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunIncrementalBuildPod(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-incremental-build"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		name := RandomizedSuffixName("java")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.strategy=pod",
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(ns, name)()
		Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		Eventually(Kit(ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
		Eventually(BuilderPodsCount(ns)).Should(Equal(1))

		t.Run("Reuse previous kit", func(t *testing.T) {
			nameClone := RandomizedSuffixName("java-clone")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameClone,
				"-t", "builder.strategy=pod",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameClone), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameClone, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameClone), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationCloneKitName := IntegrationKit(ns, nameClone)()
			Eventually(integrationCloneKitName).Should(Equal(integrationKitName))
			Eventually(BuilderPodsCount(ns)).Should(Equal(1))
		})

		t.Run("Create incremental kit", func(t *testing.T) {
			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameIncremental,
				"-d", "camel:zipfile",
				"-t", "builder.strategy=pod",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(ns, nameIncremental)()
			// the container comes in a format like
			// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
			// we should be save just to check the substring is contained
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
			Eventually(BuilderPodsCount(ns)).Should(Equal(2))
		})

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunIncrementalBuildOff(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-standard-build"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		name := RandomizedSuffixName("java")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(ns, name)()
		Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))

		t.Run("Don't reuse previous kit", func(t *testing.T) {
			nameClone := RandomizedSuffixName("java-clone")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameClone,
				"-t", "builder.incremental-image-build=false",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameClone), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameClone, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameClone), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationCloneKitName := IntegrationKit(ns, nameClone)()
			Eventually(Kit(ns, integrationCloneKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		})

		t.Run("Don't create incremental kit", func(t *testing.T) {
			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameIncremental,
				"-d", "camel:zipfile",
				"-t", "builder.incremental-image-build=false",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(ns, nameIncremental)()
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		})

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunIncrementalBuildWithDifferentBaseImages(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-standard-build"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		name := RandomizedSuffixName("java")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(ns, name)()
		Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		Eventually(Kit(ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		t.Run("Create incremental kit", func(t *testing.T) {
			// Another integration that should be built on top of the previous IntegrationKit
			// just add a new random dependency
			nameIncremental := RandomizedSuffixName("java-incremental")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", nameIncremental,
				"-d", "camel:zipfile",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, nameIncremental), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, nameIncremental, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, nameIncremental), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationIncrementalKitName := IntegrationKit(ns, nameIncremental)()
			// the container comes in a format like
			// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
			// we should be save just to check the substring is contained
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
			Eventually(Kit(ns, integrationIncrementalKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))
		})

		t.Run("Create new hierarchy kit", func(t *testing.T) {
			// We should spin off a new hierarchy of builds
			newBaseImage := "eclipse-temurin:17.0.8.1_1-jdk-ubi9-minimal"
			name = RandomizedSuffixName("java-new")
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", name,
				"-d", "camel:mongodb",
				"-t", fmt.Sprintf("builder.base-image=%s", newBaseImage),
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationKitName = IntegrationKit(ns, name)()
			Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(newBaseImage))
			Eventually(Kit(ns, integrationKitName)().Status.RootImage).Should(Equal(newBaseImage))
		})
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
