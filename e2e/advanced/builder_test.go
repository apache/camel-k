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
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuilderTimeout(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, ctx, g, ns)

		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		pl := Platform(t, ctx, ns)()
		// set a short timeout to simulate the build timeout
		pl.Spec.Build.Timeout = &metav1.Duration{
			Duration: 10 * time.Second,
		}
		TestClient(t).Update(ctx, pl)
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformTimeout(t, ctx, ns)).Should(Equal(
			&metav1.Duration{
				Duration: 10 * time.Second,
			},
		))

		operatorPod := OperatorPod(t, ctx, ns)()
		operatorPodImage := operatorPod.Spec.Containers[0].Image

		t.Run("run yaml", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml", "--name", name, "-t", "builder.strategy=pod").Execute()).To(Succeed())
			// As the build hits timeout, it keeps trying building
			g.Eventually(IntegrationPhase(t, ctx, ns, name)).Should(Equal(v1.IntegrationPhaseBuildingKit))
			integrationKitName := IntegrationKit(t, ctx, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuilderPodPhase(t, ctx, ns, builderKitName)).Should(Equal(corev1.PodPending))
			g.Eventually(BuildPhase(t, ctx, ns, integrationKitName)).Should(Equal(v1.BuildPhaseRunning))
			g.Eventually(BuilderPod(t, ctx, ns, builderKitName)().Spec.InitContainers[0].Name).Should(Equal("builder"))
			g.Eventually(BuilderPod(t, ctx, ns, builderKitName)().Spec.InitContainers[0].Image).Should(Equal(operatorPodImage))
			// After a few minutes (5 max retries), this has to be in error state
			g.Eventually(BuildPhase(t, ctx, ns, integrationKitName), TestTimeoutMedium).Should(Equal(v1.BuildPhaseError))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(BuildFailureRecoveryAttempt(t, ctx, ns, integrationKitName), TestTimeoutMedium).Should(Equal(5))
			g.Eventually(BuilderPodPhase(t, ctx, ns, builderKitName), TestTimeoutMedium).Should(Equal(corev1.PodFailed))
		})
	})
}

func TestMavenProfile(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, ctx, g, ns)
		t.Run("Run maven profile", func(t *testing.T) {
			name := RandomizedSuffixName("java-maven-profile")

			mavenProfile1Cm := newMavenProfileConfigMap(ns, "maven-profile-owasp", "owasp-profile")
			g.Expect(TestClient(t).Create(ctx, mavenProfile1Cm)).To(Succeed())
			mavenProfile2Cm := newMavenProfileConfigMap(ns, "maven-profile-dependency", "dependency-profile")
			g.Expect(TestClient(t).Create(ctx, mavenProfile2Cm)).To(Succeed())

			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "builder.maven-profiles=configmap:maven-profile-owasp/owasp-profile", "-t", "builder.maven-profiles=configmap:maven-profile-dependency/dependency-profile", "-t", "builder.tasks=custom1;alpine;cat maven/pom.xml", "-t", "builder.strategy=pod").Execute()).To(Succeed())

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

			g.Expect(TestClient(t).Delete(ctx, mavenProfile1Cm)).To(Succeed())
			g.Expect(TestClient(t).Delete(ctx, mavenProfile2Cm)).To(Succeed())
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
