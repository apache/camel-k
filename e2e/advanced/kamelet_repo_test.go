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
)

func TestKameletFromCustomRepository(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		kameletName := "timer-custom-source"
		removeKamelet(t, ctx, kameletName, ns)

		g.Eventually(Kamelet(t, ctx, kameletName, ns)).Should(BeNil())
		// Add the custom repository
		g.Expect(Kamel(t, ctx, "kamelet", "add-repo", "github:squakez/ck-kamelet-test-repo/kamelets", "-n", ns, "-x", operatorID).Execute()).To(Succeed())

		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/TimerCustomKameletIntegration.java").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "timer-custom-kamelet-integration"), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "timer-custom-kamelet-integration")).Should(ContainSubstring("hello world"))

		// Remove the custom repository
		g.Expect(Kamel(t, ctx, "kamelet", "remove-repo", "github:squakez/ck-kamelet-test-repo/kamelets", "-n", ns, "-x", operatorID).Execute()).To(Succeed())
	})
}

func removeKamelet(t *testing.T, ctx context.Context, name string, ns string) {
	kamelet := Kamelet(t, ctx, name, ns)()
	TestClient(t).Delete(ctx, kamelet)
}
