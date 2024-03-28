//go:build integration && !high_memory
// +build integration,!high_memory

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

package native

import (
	"context"
	"testing"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestNativeBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-native-binding"
		g.Expect(KamelInstallWithIDAndKameletCatalog(t, ctx, operatorID, ns, "--build-timeout", "90m0s", "--maven-cli-option", "-Dquarkus.native.native-image-xmx=6g")).To(Succeed())
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		message := "Magicstring!"
		t.Run("binding with native build", func(t *testing.T) {
			bindingName := "native-binding"
			g.Expect(KamelBindWithID(t, ctx, operatorID, ns, "timer-source", "log-sink", "-p", "source.message="+message, "--annotation", "trait.camel.apache.org/quarkus.build-mode=native", "--annotation", "trait.camel.apache.org/builder.tasks-limit-memory=quarkus-native:6.5Gi", "--name", bindingName).Execute()).To(Succeed())

			// ====================================
			// !!! THE MOST TIME-CONSUMING PART !!!
			// ====================================
			g.Eventually(Kits(t, ctx, ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutVeryLong).Should(HaveLen(1))

			nativeKit := Kits(t, ctx, ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			g.Eventually(IntegrationKit(t, ctx, ns, bindingName), TestTimeoutShort).Should(Equal(nativeKit.Name))

			g.Eventually(IntegrationPodPhase(t, ctx, ns, bindingName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, bindingName), TestTimeoutShort).Should(ContainSubstring(message))

			g.Eventually(IntegrationPod(t, ctx, ns, bindingName), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(),
					MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))

			// Clean up
			g.Expect(CamelK(t, ctx, "delete", bindingName, "-n", ns).Execute()).To(Succeed())
			g.Expect(DeleteKits(t, ctx, ns)).To(Succeed())
		})
	})
}
