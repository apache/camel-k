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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

// Tests on integration with kamelets containing configuration from properties and secrets
//  without having to change the integration code.
func TestKameletImplicitConfig(t *testing.T) {
	RegisterTestingT(t)
	t.Run("test custom timer source", func(t *testing.T) {
		Expect(CreateTimerKamelet(ns, "my-own-timer-source")()).To(Succeed())

		t.Run("run test default config using properties", func(t *testing.T) {
			name := "my-own-timer-source-config-properties"

			Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration.java",
				"-p", "camel.kamelet.my-own-timer-source.message='My Default message'",
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My Default message"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run test default config using labeled secret", func(t *testing.T) {
			name := "my-own-timer-source-default-config-secrets"
			secretName := "my-own-timer-source-default"

			var secData = make(map[string]string)
			secData["camel.kamelet.my-own-timer-source.message"] = "very top secret message"
			var labels = make(map[string]string)
			labels["camel.apache.org/kamelet"] = "my-own-timer-source"
			Expect(CreatePlainTextSecretWithLabels(ns, secretName, secData, labels)).To(Succeed())

			Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration.java",
				"-p", "camel.kamelet.my-own-timer-source.message='Default message'",
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top secret message"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
			Expect(DeleteSecret(ns, secretName)).To(Succeed())
		})

		t.Run("run test default config using mounted secret", func(t *testing.T) {
			name := "my-own-timer-source-default-config-mounted-secrets"
			secretName := "my-mounted-default-secret"

			var secData = make(map[string]string)
			secData["camel.kamelet.my-own-timer-source.message"] = "very top mounted secret message"
			Expect(CreatePlainTextSecret(ns, secretName, secData)).To(Succeed())

			Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration.java",
				"-p", "camel.kamelet.my-own-timer-source.message='Default message'",
				"-t", "mount.configs=secret:"+secretName,
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top mounted secret message"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
			Expect(DeleteSecret(ns, secretName)).To(Succeed())
		})

		t.Run("run test named config using properties", func(t *testing.T) {
			name := "my-own-timer-source-config-properties"
			Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration.java",
				"-p", "camel.kamelet.my-own-timer-source.message='Default message'",
				"-p", "camel.kamelet.my-own-timer-source.mynamedconfig.message='My Named Config message'",
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My Named Config message"))
			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run test named config using labeled secrets", func(t *testing.T) {
			name := "my-own-timer-source-implicit-config-secrets"
			secretName := "my-own-timer-source-mynamedconfig"

			var secData = make(map[string]string)
			secData["camel.kamelet.my-own-timer-source.mynamedconfig.message"] = "very top named secret message"
			var labels = make(map[string]string)
			labels["camel.apache.org/kamelet"] = "my-own-timer-source"
			labels["camel.apache.org/kamelet.configuration"] = "mynamedconfig"
			Expect(CreatePlainTextSecretWithLabels(ns, secretName, secData, labels)).To(Succeed())

			Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration.java",
				"-p", "camel.kamelet.my-own-timer-source.message='Default message'",
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top named secret message"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
			Expect(DeleteSecret(ns, secretName)).To(Succeed())
		})

		t.Run("run test named config using mounted secrets", func(t *testing.T) {
			name := "my-own-timer-source-named-config-mounted-secrets"
			secretName := "my-mounted-named-secret"

			var secData = make(map[string]string)
			secData["camel.kamelet.my-own-timer-source.mynamedconfig.message"] = "very top named mounted secret message"
			Expect(CreatePlainTextSecret(ns, secretName, secData)).To(Succeed())

			Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration.java",
				"-p", "camel.kamelet.my-own-timer-source.message='Default message'",
				"-t", "mount.configs=secret:"+secretName,
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top named mounted secret message"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
			Expect(DeleteSecret(ns, secretName)).To(Succeed())
		})

	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	Expect(DeleteKamelet(ns, "my-own-timer-source")).To(Succeed())
}

// Tests on integration with kamelets containing configuration from properties and secrets with parameters inside the integration.
func TestKameletConfig(t *testing.T) {
	RegisterTestingT(t)
	t.Run("test custom timer source", func(t *testing.T) {
		Expect(CreateTimerKamelet(ns, "my-own-timer-source")()).To(Succeed())
		Expect(CreateLogKamelet(ns, "my-own-log-sink")()).To(Succeed())
		t.Run("run test default config inlined properties", func(t *testing.T) {
			name := "my-own-timer-source-inline-properties"

			Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-inlined-configuration.yaml",
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("important message"))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("integrationLogger"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run test default config parameters properties", func(t *testing.T) {
			name := "my-own-timer-source-parameters-properties"

			Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-parameters-configuration.yaml",
				"-p", "my-message='My parameter message'",
				"-p", "my-logger='myIntegrationLogger'",
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My parameter message"))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("myIntegrationLogger"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run test default config secrer properties", func(t *testing.T) {
			name := "my-own-timer-source-secret-properties"
			secretName := "my-mounted-secret-properties"

			var secData = make(map[string]string)
			secData["my-message"] = "My secret message"
			secData["my-logger"] = "mySecretIntegrationLogger"
			Expect(CreatePlainTextSecret(ns, secretName, secData)).To(Succeed())

			Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-parameters-configuration.yaml",
				"-t", "mount.configs=secret:"+secretName,
				"--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My secret message"))
			Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("mySecretIntegrationLogger"))

			Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
			Expect(DeleteSecret(ns, secretName)).To(Succeed())
		})

	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	Expect(DeleteKamelet(ns, "my-own-timer-source")).To(Succeed())
	Expect(DeleteKamelet(ns, "my-own-log-sink")).To(Succeed())
}
