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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuilderTimeout(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		g.Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		g.Eventually(Platform(t, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformConditionStatus(t, ns, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))

		pl := Platform(t, ns)()
		// set a short timeout to simulate the build timeout
		pl.Spec.Build.Timeout = &metav1.Duration{
			Duration: 10 * time.Second,
		}
		TestClient(t).Update(TestContext, pl)
		g.Eventually(Platform(t, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformTimeout(t, ns)).Should(Equal(
			&metav1.Duration{
				Duration: 10 * time.Second,
			},
		))

		operatorPod := OperatorPod(t, ns)()
		operatorPodImage := operatorPod.Spec.Containers[0].Image

		t.Run("run yaml", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml",
				"--name", name,
				"-t", "builder.strategy=pod").Execute()).To(Succeed())
			// As the build hits timeout, it keeps trying building
			g.Eventually(IntegrationPhase(t, ns, name)).Should(Equal(v1.IntegrationPhaseBuildingKit))
			integrationKitName := IntegrationKit(t, ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			g.Eventually(BuilderPodPhase(t, ns, builderKitName)).Should(Equal(corev1.PodPending))
			g.Eventually(BuildPhase(t, ns, integrationKitName)).Should(Equal(v1.BuildPhaseRunning))
			g.Eventually(BuilderPod(t, ns, builderKitName)().Spec.InitContainers[0].Name).Should(Equal("builder"))
			g.Eventually(BuilderPod(t, ns, builderKitName)().Spec.InitContainers[0].Image).Should(Equal(operatorPodImage))
			// After a few minutes (5 max retries), this has to be in error state
			g.Eventually(BuildPhase(t, ns, integrationKitName), TestTimeoutMedium).Should(Equal(v1.BuildPhaseError))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(BuildFailureRecovery(t, ns, integrationKitName), TestTimeoutMedium).Should(Equal(5))
			g.Eventually(BuilderPodPhase(t, ns, builderKitName), TestTimeoutMedium).Should(Equal(corev1.PodFailed))
		})
	})
}
