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
)

func TestKameletFromCustomRepository(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, ctx, g, ns)

		kameletName := "timer-custom-source"
		removeKamelet(t, ctx, kameletName, ns)
		g.Eventually(Kamelet(t, ctx, kameletName, ns)).Should(BeNil())

		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		pl := Platform(t, ctx, ns)()
		pl.Spec.Kamelet.Repositories = []v1.KameletRepositorySpec{
			{URI: "github:squakez/ck-kamelet-test-repo/kamelets"},
		}
		TestClient(t).Update(ctx, pl)
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformHas(t, ctx, ns, func(pl *v1.IntegrationPlatform) bool {
			return len(pl.Status.Kamelet.Repositories) > 0 && pl.Status.Kamelet.Repositories[0].URI == "github:squakez/ck-kamelet-test-repo/kamelets"
		}), TestTimeoutShort).Should(BeTrue())

		// Add the custom repository
		g.Expect(KamelRun(t, ctx, ns, "files/TimerCustomKameletIntegration.java").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "timer-custom-kamelet-integration"), TestTimeoutMedium).
			Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "timer-custom-kamelet-integration")).Should(ContainSubstring("hello world"))
	})
}

func removeKamelet(t *testing.T, ctx context.Context, name string, ns string) {
	kamelet := Kamelet(t, ctx, name, ns)()
	TestClient(t).Delete(ctx, kamelet)
}
