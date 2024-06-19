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

package config

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

// Tests on integrations with kamelets containing configuration from properties and secrets
// without having to change the integration code.
func TestKameletImplicitConfigDefaultUserProperty(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("run test default config using properties", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig01-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int01")
			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationConfiguration01.java",
				"-p", "camel.kamelet.iconfig01-timer-source.message='Default message 01'",
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("Default message 01"))
		})

		t.Run("run test default config using mounted secret", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig03-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int3")
			secretName := "my-iconfig-int3-secret"

			var secData = make(map[string]string)
			secData["camel.kamelet.iconfig03-timer-source.message"] = "very top mounted secret message"
			g.Expect(CreatePlainTextSecret(t, ctx, ns, secretName, secData)).To(Succeed())
			g.Eventually(SecretByName(t, ctx, ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationConfiguration03.java",
				"-t", "mount.configs=secret:"+secretName,
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("very top mounted secret message"))
		})

		t.Run("run test default config using mounted configmap", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig04-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int4")
			cmName := "my-iconfig-int4-configmap"

			var cmData = make(map[string]string)
			cmData["camel.kamelet.iconfig04-timer-source.message"] = "very top mounted configmap message"
			g.Expect(CreatePlainTextConfigmap(t, ctx, ns, cmName, cmData)).To(Succeed())

			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationConfiguration04.java",
				"-t", "mount.configs=configmap:"+cmName,
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("very top mounted configmap message"))
		})

		t.Run("run test named config using properties", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig05-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int5")
			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationNamedConfiguration05.java",
				"-p", "camel.kamelet.iconfig05-timer-source.message='Default message 05'",
				"-p", "camel.kamelet.iconfig05-timer-source.mynamedconfig.message='My Named Config message'",
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My Named Config message"))
		})

		t.Run("run test named config using labeled secret", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig06-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int6")
			secretName := "my-iconfig-int6-secret"

			var secData = make(map[string]string)
			secData["camel.kamelet.iconfig06-timer-source.mynamedconfig.message"] = "very top named secret message"
			var labels = make(map[string]string)
			labels["camel.apache.org/kamelet"] = "iconfig06-timer-source"
			labels["camel.apache.org/kamelet.configuration"] = "mynamedconfig"
			g.Expect(CreatePlainTextSecretWithLabels(t, ctx, ns, secretName, secData, labels)).To(Succeed())
			g.Eventually(SecretByName(t, ctx, ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationNamedConfiguration06.java",
				"-p", "camel.kamelet.iconfig06-timer-source.message='Default message 06'",
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("very top named secret message"))
		})

		t.Run("run test named config using mounted secret", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig07-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int7")
			secretName := "my-iconfig-int7-secret"

			var secData = make(map[string]string)
			secData["camel.kamelet.iconfig07-timer-source.mynamedconfig.message"] = "very top named mounted secret message"
			g.Expect(CreatePlainTextSecret(t, ctx, ns, secretName, secData)).To(Succeed())
			g.Eventually(SecretByName(t, ctx, ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationNamedConfiguration07.java",
				"-p", "camel.kamelet.iconfig07-timer-source.message='Default message 07'",
				"-t", "mount.configs=secret:"+secretName,
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("very top named mounted secret message"))
		})

		t.Run("run test named config using mounted configmap", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig08-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int8")
			cmName := "my-iconfig-int8-configmap"

			var cmData = make(map[string]string)
			cmData["camel.kamelet.iconfig08-timer-source.mynamedconfig.message"] = "very top named mounted configmap message"
			g.Expect(CreatePlainTextConfigmap(t, ctx, ns, cmName, cmData)).To(Succeed())

			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationNamedConfiguration08.java",
				"-p", "camel.kamelet.iconfig08-timer-source.message='Default message 08'",
				"-t", "mount.configs=configmap:"+cmName,
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("very top named mounted configmap message"))
		})

		t.Run("run test default config using labeled secret", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "iconfig09-timer-source")()).To(Succeed())

			name := RandomizedSuffixName("iconfig-test-timer-source-int9")
			secretName := "my-iconfig-int9-secret"

			var secData = make(map[string]string)
			secData["camel.kamelet.iconfig09-timer-source.message"] = "very top labeled secret message"
			var labels = make(map[string]string)
			labels["camel.apache.org/kamelet"] = "iconfig09-timer-source"
			g.Expect(CreatePlainTextSecretWithLabels(t, ctx, ns, secretName, secData, labels)).To(Succeed())
			g.Eventually(SecretByName(t, ctx, ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

			g.Expect(KamelRun(t, ctx, ns, "files/TimerKameletIntegrationConfiguration09.java",
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("very top labeled secret message"))
		})

		t.Run("run test default config inlined properties", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "config01-timer-source")()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, "config01-log-sink")()).To(Succeed())

			name := RandomizedSuffixName("config-test-timer-source-int1")

			g.Expect(KamelRun(t, ctx, ns, "files/timer-kamelet-integration-inlined-configuration-01.yaml",
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("important message"))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("integrationLogger"))
		})

		t.Run("run test default config parameters properties", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "config02-timer-source")()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, "config02-log-sink")()).To(Succeed())

			name := RandomizedSuffixName("config-test-timer-source-int2")

			g.Expect(KamelRun(t, ctx, ns, "files/timer-kamelet-integration-parameters-configuration-02.yaml",
				"-p", "my-message='My parameter message 02'",
				"-p", "my-logger='myIntegrationLogger02'",
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My parameter message 02"))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myIntegrationLogger02"))
		})

		t.Run("run test default config secret properties", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "config03-timer-source")()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, "config03-log-sink")()).To(Succeed())

			name := RandomizedSuffixName("config-test-timer-source-int3")
			secretName := "my-config-int3-secret"

			var secData = make(map[string]string)
			secData["my-message"] = "My secret message 03"
			secData["my-logger"] = "mySecretIntegrationLogger03"
			g.Expect(CreatePlainTextSecret(t, ctx, ns, secretName, secData)).To(Succeed())

			g.Expect(KamelRun(t, ctx, ns, "files/timer-kamelet-integration-parameters-configuration-03.yaml",
				"-t", "mount.configs=secret:"+secretName,
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My secret message 03"))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("mySecretIntegrationLogger03"))
		})

		t.Run("run test default config configmap properties", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "config04-timer-source")()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, "config04-log-sink")()).To(Succeed())

			name := RandomizedSuffixName("config-test-timer-source-int4")
			cmName := "my-config-int4-configmap"

			var cmData = make(map[string]string)
			cmData["my-message"] = "My configmap message 04"
			cmData["my-logger"] = "myConfigmapIntegrationLogger04"
			g.Expect(CreatePlainTextConfigmap(t, ctx, ns, cmName, cmData)).To(Succeed())

			g.Expect(KamelRun(t, ctx, ns, "files/timer-kamelet-integration-parameters-configuration-04.yaml",
				"-t", "mount.configs=configmap:"+cmName,
				"--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My configmap message 04"))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myConfigmapIntegrationLogger04"))
		})

	})
}
