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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuilderTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-traits-builder"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("Run build strategy routine", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "builder.order-strategy=sequential", "-t", "builder.strategy=routine").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyRoutine))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().OrderStrategy, TestTimeoutShort).Should(Equal(v1.BuildOrderStrategySequential))
			// Default resource CPU Check
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal(""))

			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName), TestTimeoutShort).Should(BeNil())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Run build order strategy dependencies", func(t *testing.T) {
			name := RandomizedSuffixName("java-dependencies-strategy")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java",
				"--name", name,
				// This is required in order to avoid reusing a Kit already existing (which is the default behavior)
				"--build-property", "strategy=dependencies",
				"-t", "builder.order-strategy=dependencies").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyRoutine))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().OrderStrategy, TestTimeoutShort).Should(Equal(v1.BuildOrderStrategyDependencies))
			// Default resource CPU Check
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName), TestTimeoutShort).Should(BeNil())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Run build order strategy fifo", func(t *testing.T) {
			name := RandomizedSuffixName("java-fifo-strategy")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java",
				"--name", name,
				// This is required in order to avoid reusing a Kit already existing (which is the default behavior)
				"--build-property", "strategy=fifo",
				"-t", "builder.order-strategy=fifo").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyRoutine))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().OrderStrategy, TestTimeoutShort).Should(Equal(v1.BuildOrderStrategyFIFO))
			// Default resource CPU Check
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal(""))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal(""))

			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName), TestTimeoutShort).Should(BeNil())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Run build resources configuration", func(t *testing.T) {
			name := RandomizedSuffixName("java-resource-config")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java",
				"--name", name,
				// This is required in order to avoid reusing a Kit already existing (which is the default behavior)
				"--build-property", "resources=new-build",
				"-t", "builder.tasks-request-cpu=builder:500m",
				"-t", "builder.tasks-limit-cpu=builder:1000m",
				"-t", "builder.tasks-request-memory=builder:2Gi",
				"-t", "builder.tasks-limit-memory=builder:3Gi",
				"-t", "builder.strategy=pod",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)

			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyPod))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().OrderStrategy, TestTimeoutShort).Should(Equal(v1.BuildOrderStrategySequential))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal("500m"))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal("1000m"))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal("2Gi"))
			g.Eventually(BuildConfig(t, ctx, integrationKitNamespace, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal("3Gi"))

			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName), TestTimeoutShort).ShouldNot(BeNil())
			// Let's assert we set the resources on the builder container
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Name, TestTimeoutShort).Should(Equal("builder"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Resources.Requests.Cpu().String(), TestTimeoutShort).Should(Equal("500m"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Resources.Limits.Cpu().String(), TestTimeoutShort).Should(Equal("1"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Resources.Requests.Memory().String(), TestTimeoutShort).Should(Equal("2Gi"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Resources.Limits.Memory().String(), TestTimeoutShort).Should(Equal("3Gi"))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Run custom pipeline task", func(t *testing.T) {
			name := RandomizedSuffixName("java-pipeline")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "builder.tasks=custom1;alpine;tree", "-t", "builder.tasks=custom2;alpine;cat maven/pom.xml", "-t", "builder.strategy=pod").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(len(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers), TestTimeoutShort).Should(Equal(4))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Name, TestTimeoutShort).Should(Equal("builder"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[1].Name, TestTimeoutShort).Should(Equal("custom1"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[2].Name, TestTimeoutShort).Should(Equal("custom2"))

			// Check containers conditions
			g.Eventually(Build(t, ctx, integrationKitNamespace, integrationKitName), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(
				Build(t, ctx, integrationKitNamespace, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Containercustom1Succeeded")).Status,
				TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(
				Build(t, ctx, integrationKitNamespace, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Containercustom1Succeeded")).Message,
				TestTimeoutShort).Should(ContainSubstring("generated-bytecode.jar"))
			g.Eventually(Build(t, ctx, integrationKitNamespace, integrationKitName), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(
				Build(t, ctx, integrationKitNamespace, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Containercustom2Succeeded")).Status,
				TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(
				Build(t, ctx, integrationKitNamespace, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Containercustom2Succeeded")).Message,
				TestTimeoutShort).Should(ContainSubstring("</project>"))

			// Check logs
			g.Eventually(Logs(t, ctx, integrationKitNamespace, builderKitName, corev1.PodLogOptions{Container: "custom1"})).Should(ContainSubstring(`generated-bytecode.jar`))
			g.Eventually(Logs(t, ctx, integrationKitNamespace, builderKitName, corev1.PodLogOptions{Container: "custom2"})).Should(ContainSubstring(`<artifactId>camel-k-runtime-bom</artifactId>`))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Run custom pipeline task error", func(t *testing.T) {
			name := RandomizedSuffixName("java-error")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "builder.tasks=custom1;alpine;cat missingfile.txt", "-t", "builder.strategy=pod").Execute()).To(Succeed())

			g.Eventually(IntegrationPhase(t, ctx, ns, name)).Should(Equal(v1.IntegrationPhaseBuildingKit))
			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			// Check containers conditions
			g.Eventually(Build(t, ctx, integrationKitNamespace, integrationKitName), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(BuildConditions(t, ctx, integrationKitNamespace, integrationKitName), TestTimeoutLong).ShouldNot(BeNil())
			g.Eventually(BuildCondition(t, ctx, integrationKitNamespace, integrationKitName, v1.BuildConditionType("Containercustom1Succeeded")), TestTimeoutMedium).ShouldNot(BeNil())
			g.Eventually(
				BuildCondition(t, ctx, integrationKitNamespace, integrationKitName, v1.BuildConditionType("Containercustom1Succeeded"))().Status,
				TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
			g.Eventually(
				BuildCondition(t, ctx, integrationKitNamespace, integrationKitName, v1.BuildConditionType("Containercustom1Succeeded"))().Message,
				TestTimeoutShort).Should(ContainSubstring("No such file or directory"))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Run maven profile", func(t *testing.T) {
			name := RandomizedSuffixName("java-maven-profile")

			mavenProfile1Cm := newMavenProfileConfigMap(ns, "maven-profile-owasp", "owasp-profile")
			g.Expect(TestClient(t).Create(ctx, mavenProfile1Cm)).To(Succeed())
			mavenProfile2Cm := newMavenProfileConfigMap(ns, "maven-profile-dependency", "dependency-profile")
			g.Expect(TestClient(t).Create(ctx, mavenProfile2Cm)).To(Succeed())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "builder.maven-profiles=configmap:maven-profile-owasp/owasp-profile", "-t", "builder.maven-profiles=configmap:maven-profile-dependency/dependency-profile", "-t", "builder.tasks=custom1;alpine;cat maven/pom.xml", "-t", "builder.strategy=pod").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(len(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers), TestTimeoutShort).Should(Equal(3))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[0].Name, TestTimeoutShort).Should(Equal("builder"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[1].Name, TestTimeoutShort).Should(Equal("custom1"))
			g.Eventually(BuilderPod(t, ctx, integrationKitNamespace, builderKitName)().Spec.InitContainers[2].Name, TestTimeoutShort).Should(Equal("package"))

			// Check containers conditions
			g.Eventually(Build(t, ctx, integrationKitNamespace, integrationKitName), TestTimeoutShort).ShouldNot(BeNil())
			g.Eventually(
				Build(t, ctx, integrationKitNamespace, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Containercustom1Succeeded")).Status,
				TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(
				Build(t, ctx, integrationKitNamespace, integrationKitName)().Status.GetCondition(v1.BuildConditionType("Containercustom1Succeeded")).Message,
				TestTimeoutShort).Should(ContainSubstring("</project>"))

			// Check logs
			g.Eventually(Logs(t, ctx, integrationKitNamespace, builderKitName, corev1.PodLogOptions{Container: "custom1"})).Should(ContainSubstring(`<id>owasp-profile</id>`))
			g.Eventually(Logs(t, ctx, integrationKitNamespace, builderKitName, corev1.PodLogOptions{Container: "custom1"})).Should(ContainSubstring(`<id>dependency-profile</id>`))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Expect(TestClient(t).Delete(ctx, mavenProfile1Cm)).To(Succeed())
			g.Expect(TestClient(t).Delete(ctx, mavenProfile2Cm)).To(Succeed())
		})

		t.Run("Run distroless container image", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "builder.base-image=gcr.io/distroless/java17-debian12").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			g.Eventually(KitRootImage(t, ctx, ns, integrationKitName), TestTimeoutShort).Should(Equal("gcr.io/distroless/java17-debian12"))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}

func newMavenProfileConfigMap(ns, name, key string) *corev1.ConfigMap {
	return &corev1.ConfigMap{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ConfigMap",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
		Data: map[string]string{
			key: fmt.Sprintf(`
<profile>
  <id>` + key + `</id>
  <build>
    <plugins>
      <plugin>
        <groupId>org.owasp</groupId>
        <artifactId>dependency-check-maven</artifactId>
        <version>5.3.0</version>
        <executions>
          <execution>
            <goals>
              <goal>check</goal>
            </goals>
          </execution>
        </executions>
      </plugin>
    </plugins>
  </build>
</profile>
`,
			),
		},
	}
}
