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
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformConditionStatus(t, ctx, ns, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))

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
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml", "--name", name, "-t", "builder.strategy=pod").Execute()).To(Succeed())
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
			g.Eventually(BuildFailureRecovery(t, ctx, ns, integrationKitName), TestTimeoutMedium).Should(Equal(5))
			g.Eventually(BuilderPodPhase(t, ctx, ns, builderKitName), TestTimeoutMedium).Should(Equal(corev1.PodFailed))
		})
	})
}
