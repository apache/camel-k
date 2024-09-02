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
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestNativeIntegrations(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("unsupported integration source language", func(t *testing.T) {
			name := RandomizedSuffixName("unsupported-js")
			g.Expect(KamelRun(t, ctx, ns, "files/JavaScript.js", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi").Execute()).To(Succeed())

			g.Eventually(IntegrationPhase(t, ctx, ns, name)).Should(Equal(v1.IntegrationPhaseError))
		})

		t.Run("xml native support", func(t *testing.T) {
			name := RandomizedSuffixName("xml-native")
			g.Expect(KamelRun(t, ctx, ns, "files/Xml.xml", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("XML Magicstring!"))

			g.Eventually(IntegrationKitLayout(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationKitLayoutNativeSources))
		})

		t.Run("automatic rollout deployment from jvm to native kit", func(t *testing.T) {
			name := RandomizedSuffixName("yaml-native")
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml", "--name", name, "-t", "quarkus.build-mode=jvm", "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi").Execute()).To(Succeed())

			// Check the fast-jar Kit is ready
			g.Eventually(IntegrationKitLayout(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationKitLayoutFastJar))
			g.Eventually(IntegrationKitStatusPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationKitPhaseReady))

			// Check the Integration is ready
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), ContainSubstring("java")))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))

			// Check the native Kit is ready
			g.Eventually(IntegrationKitLayout(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(v1.IntegrationKitLayoutNativeSources))
			g.Eventually(IntegrationKitStatusPhase(t, ctx, ns, name)).Should(Equal(v1.IntegrationKitPhaseReady))

			// Check the Integration is still ready
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			t.Run("yaml native should not rebuild", func(t *testing.T) {
				name := RandomizedSuffixName("yaml-native-2")
				g.Expect(KamelRun(t, ctx, ns, "files/yaml2.yaml", "--name", name, "-t", "quarkus.build-mode=native", "-t", "builder.tasks-limit-memory=quarkus-native:6.5Gi").Execute()).To(Succeed())

				// This one should run quickly as it suppose to reuse an IntegrationKit
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutShort).
					Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!2"))
				g.Eventually(IntegrationKit(t, ctx, ns, "yaml-native-2")).Should(Equal(IntegrationKit(t, ctx, ns, "yaml-native")()))
			})
		})

	})
}

func TestNativeBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		message := "Magicstring!"
		t.Run("binding with native build", func(t *testing.T) {
			bindingName := "native-binding"
			g.Expect(KamelBind(t, ctx, ns, "timer-source", "log-sink", "-p", "source.message="+message,
				"--trait", "quarkus.build-mode=native", "--trait", "builder.tasks-limit-memory=quarkus-native:6.5Gi",
				"--name", bindingName).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, bindingName), TestTimeoutVeryLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, bindingName, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, bindingName), TestTimeoutShort).Should(ContainSubstring(message))
			g.Eventually(IntegrationPod(t, ctx, ns, bindingName), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(),
					MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))

			g.Eventually(IntegrationKitLayout(t, ctx, ns, bindingName), TestTimeoutShort).Should(Equal(v1.IntegrationKitLayoutNativeSources))
		})
	})
}
