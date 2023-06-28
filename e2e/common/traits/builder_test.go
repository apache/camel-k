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

package traits

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuilderTrait(t *testing.T) {
	RegisterTestingT(t)

	name := "java"

	t.Run("Run build strategy routine", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.strategy=routine").Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		integrationKitName := IntegrationKit(ns, name)()
		builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
		Eventually(BuildConfig(ns, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyRoutine))
		// Default resource CPU Check
		Eventually(BuildConfig(ns, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal(""))
		Eventually(BuildConfig(ns, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal(""))
		Eventually(BuildConfig(ns, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal(""))
		Eventually(BuildConfig(ns, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal(""))

		Eventually(BuilderPod(ns, builderKitName), TestTimeoutShort).Should(BeNil())

		// We need to remove the kit as well
		Expect(Kamel("reset", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Run build resources configuration", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.request-cpu=500m",
			"-t", "builder.limit-cpu=1000m",
			"-t", "builder.request-memory=2Gi",
			"-t", "builder.limit-memory=3Gi",
			"-t", "builder.strategy=pod",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		integrationKitName := IntegrationKit(ns, name)()
		builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)

		Eventually(BuildConfig(ns, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyPod))
		Eventually(BuildConfig(ns, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal("500m"))
		Eventually(BuildConfig(ns, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal("1000m"))
		Eventually(BuildConfig(ns, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal("2Gi"))
		Eventually(BuildConfig(ns, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal("3Gi"))

		Eventually(BuilderPod(ns, builderKitName), TestTimeoutShort).ShouldNot(BeNil())
		// Let's assert we set the resources on the builder container
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Name, TestTimeoutShort).Should(Equal("builder"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Requests.Cpu().String(), TestTimeoutShort).Should(Equal("500m"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Limits.Cpu().String(), TestTimeoutShort).Should(Equal("1"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Requests.Memory().String(), TestTimeoutShort).Should(Equal("2Gi"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Limits.Memory().String(), TestTimeoutShort).Should(Equal("3Gi"))

		Expect(Kamel("reset", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Run custom pipeline task", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.tasks=custom1;alpine;tree",
			"-t", "builder.tasks=custom2;alpine;cat maven/pom.xml",
			"-t", "builder.strategy=pod",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		integrationKitName := IntegrationKit(ns, name)()
		builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
		Eventually(BuilderPod(ns, builderKitName), TestTimeoutShort).ShouldNot(BeNil())
		Eventually(len(BuilderPod(ns, builderKitName)().Spec.InitContainers), TestTimeoutShort).Should(Equal(3))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Name, TestTimeoutShort).Should(Equal("builder"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[1].Name, TestTimeoutShort).Should(Equal("custom1"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[2].Name, TestTimeoutShort).Should(Equal("custom2"))

		// Check containers conditions
		Eventually(Build(ns, integrationKitName), TestTimeoutShort).ShouldNot(BeNil())
		Eventually(
			Build(
				ns, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Container custom1 succeeded")).Status,
			TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(
			Build(ns, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Container custom1 succeeded")).Message,
			TestTimeoutShort).Should(ContainSubstring("generated-bytecode.jar"))
		Eventually(Build(ns, integrationKitName), TestTimeoutShort).ShouldNot(BeNil())
		Eventually(
			Build(ns, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Container custom2 succeeded")).Status,
			TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(
			Build(ns, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Container custom2 succeeded")).Message,
			TestTimeoutShort).Should(ContainSubstring("</project>"))

		// Check logs
		Eventually(Logs(ns, builderKitName, corev1.PodLogOptions{Container: "custom1"})).Should(ContainSubstring(`generated-bytecode.jar`))
		Eventually(Logs(ns, builderKitName, corev1.PodLogOptions{Container: "custom2"})).Should(ContainSubstring(`<artifactId>camel-k-runtime-bom</artifactId>`))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	name = "java-error"
	t.Run("Run custom pipeline task error", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.tasks=custom1;alpine;cat missingfile.txt",
			"-t", "builder.strategy=pod",
		).Execute()).To(Succeed())

		Eventually(IntegrationPhase(ns, name)).Should(Equal(v1.IntegrationPhaseBuildingKit))
		integrationKitName := IntegrationKit(ns, name)()
		// Check containers conditions
		Eventually(Build(ns, integrationKitName), TestTimeoutLong).ShouldNot(BeNil())
		Eventually(BuildConditions(ns, integrationKitName), TestTimeoutLong).ShouldNot(BeNil())
		Eventually(
			Build(ns, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Container custom1 succeeded")).Status,
			TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		Eventually(
			Build(ns, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Container custom1 succeeded")).Message,
			TestTimeoutShort).Should(ContainSubstring("No such file or directory"))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
