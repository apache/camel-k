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

package native

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestNativeHighMemoryIntegrations(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		javaNativeName := RandomizedSuffixName("java-native")
		javaNativeCloneName := RandomizedSuffixName("java-native-clone")
		javaNative2Name := RandomizedSuffixName("java-native-2")

		t.Run("java native support", func(t *testing.T) {
			name := javaNativeName
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Java Magicstring!"))

			t.Run("java native same should not rebuild", func(t *testing.T) {
				name := javaNativeCloneName
				g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi").Execute()).To(Succeed())

				// This one should run quickly as it suppose to reuse an IntegrationKit
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
					Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Java Magicstring!"))
				g.Eventually(IntegrationKit(t, ctx, ns, javaNativeCloneName)).Should(Equal(IntegrationKit(t, ctx, ns, javaNativeName)()))
			})

			t.Run("java native should rebuild", func(t *testing.T) {
				name := javaNative2Name
				g.Expect(KamelRun(t, ctx, ns, "files/Java2.java", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi").Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
					Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Java Magic2string!"))
				g.Eventually(IntegrationKit(t, ctx, ns, javaNative2Name)).ShouldNot(Equal(IntegrationKit(t, ctx, ns, javaNativeName)()))
			})

		})

		t.Run("groovy native support", func(t *testing.T) {
			name := RandomizedSuffixName("groovy-native")
			g.Expect(KamelRun(t, ctx, ns, "files/Groovy.groovy", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Groovy Magicstring!"))
		})

		t.Run("kotlin native support", func(t *testing.T) {
			name := RandomizedSuffixName("kotlin-native")
			g.Expect(KamelRun(t, ctx, ns, "files/Kotlin.kts", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:9.5Gi").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))

			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Kotlin Magicstring!"))
		})
	})
}
