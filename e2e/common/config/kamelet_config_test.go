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

// Tests on integrations with kamelets containing configuration from properties and secrets
//
//	without having to change the integration code.
func TestKameletImplicitConfigDefaultUserPropery(t *testing.T) {
	RegisterTestingT(t)
	t.Run("run test default config using properties", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig01-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int01"
		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration01.java",
			"-p", "camel.kamelet.iconfig01-timer-source.message='Default message 01'",
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("Default message 01"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "iconfig01-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigDefaultMountedSecret(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run test default config using mounted secret", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig03-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int3"
		secretName := "my-iconfig-int3-secret"

		var secData = make(map[string]string)
		secData["camel.kamelet.iconfig03-timer-source.message"] = "very top mounted secret message"
		Expect(CreatePlainTextSecret(ns, secretName, secData)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration03.java",
			"-t", "mount.configs=secret:"+secretName,
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top mounted secret message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteSecret(ns, secretName)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "iconfig03-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigDefaultMountedConfigmap(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run test default config using mounted configmap", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig04-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int4"
		cmName := "my-iconfig-int4-configmap"

		var cmData = make(map[string]string)
		cmData["camel.kamelet.iconfig04-timer-source.message"] = "very top mounted configmap message"
		Expect(CreatePlainTextConfigmap(ns, cmName, cmData)).To(Succeed())

		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration04.java",
			"-t", "mount.configs=configmap:"+cmName,
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top mounted configmap message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteConfigmap(ns, cmName)).To(Succeed())
		Expect(DeleteKamelet(ns, "iconfig04-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigNamedUserPropery(t *testing.T) {
	RegisterTestingT(t)
	t.Run("run test named config using properties", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig05-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int5"
		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration05.java",
			"-p", "camel.kamelet.iconfig05-timer-source.message='Default message 05'",
			"-p", "camel.kamelet.iconfig05-timer-source.mynamedconfig.message='My Named Config message'",
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My Named Config message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "iconfig05-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigNamedLabeledSecret(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run test named config using labeled secret", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig06-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int6"
		secretName := "my-iconfig-int6-secret"

		var secData = make(map[string]string)
		secData["camel.kamelet.iconfig06-timer-source.mynamedconfig.message"] = "very top named secret message"
		var labels = make(map[string]string)
		labels["camel.apache.org/kamelet"] = "iconfig06-timer-source"
		labels["camel.apache.org/kamelet.configuration"] = "mynamedconfig"
		Expect(CreatePlainTextSecretWithLabels(ns, secretName, secData, labels)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration06.java",
			"-p", "camel.kamelet.iconfig06-timer-source.message='Default message 06'",
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top named secret message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteSecret(ns, secretName)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "iconfig06-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigNamedMountedSecret(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run test named config using mounted secret", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig07-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int7"
		secretName := "my-iconfig-int7-secret"

		var secData = make(map[string]string)
		secData["camel.kamelet.iconfig07-timer-source.mynamedconfig.message"] = "very top named mounted secret message"
		Expect(CreatePlainTextSecret(ns, secretName, secData)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration07.java",
			"-p", "camel.kamelet.iconfig07-timer-source.message='Default message 07'",
			"-t", "mount.configs=secret:"+secretName,
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top named mounted secret message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteSecret(ns, secretName)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "iconfig07-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigNamedMountedConfigmap(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run test named config using mounted configmap", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig08-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int8"
		cmName := "my-iconfig-int8-configmap"

		var cmData = make(map[string]string)
		cmData["camel.kamelet.iconfig08-timer-source.mynamedconfig.message"] = "very top named mounted configmap message"
		Expect(CreatePlainTextConfigmap(ns, cmName, cmData)).To(Succeed())

		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationNamedConfiguration08.java",
			"-p", "camel.kamelet.iconfig08-timer-source.message='Default message 08'",
			"-t", "mount.configs=configmap:"+cmName,
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top named mounted configmap message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteConfigmap(ns, cmName)).To(Succeed())
		Expect(DeleteKamelet(ns, "iconfig08-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletImplicitConfigDefaultLabeledSecret(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run test default config using labeled secret", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "iconfig09-timer-source")()).To(Succeed())

		name := "iconfig-test-timer-source-int9"
		secretName := "my-iconfig-int9-secret"

		var secData = make(map[string]string)
		secData["camel.kamelet.iconfig09-timer-source.message"] = "very top labeled secret message"
		var labels = make(map[string]string)
		labels["camel.apache.org/kamelet"] = "iconfig09-timer-source"
		Expect(CreatePlainTextSecretWithLabels(ns, secretName, secData, labels)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(Not(BeNil()))

		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegrationConfiguration09.java",
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("very top labeled secret message"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteSecret(ns, secretName)).To(Succeed())
		Eventually(SecretByName(ns, secretName), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "iconfig09-timer-source")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

// Tests on integration with kamelets containing configuration from properties and secrets with parameters inside the integration.

func TestKameletConfigInlinedUserPropery(t *testing.T) {
	RegisterTestingT(t)
	t.Run("run test default config inlined properties", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "config01-timer-source")()).To(Succeed())
		Expect(CreateLogKamelet(ns, "config01-log-sink")()).To(Succeed())

		name := "config-test-timer-source-int1"

		Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-inlined-configuration-01.yaml",
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("important message"))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("integrationLogger"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "config01-timer-source")).To(Succeed())
		Expect(DeleteKamelet(ns, "config01-log-sink")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletConfigDefaultParamUserPropery(t *testing.T) {
	RegisterTestingT(t)
	t.Run("run test default config parameters properties", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "config02-timer-source")()).To(Succeed())
		Expect(CreateLogKamelet(ns, "config02-log-sink")()).To(Succeed())

		name := "config-test-timer-source-int2"

		Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-parameters-configuration-02.yaml",
			"-p", "my-message='My parameter message 02'",
			"-p", "my-logger='myIntegrationLogger02'",
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My parameter message 02"))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("myIntegrationLogger02"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteKamelet(ns, "config02-timer-source")).To(Succeed())
		Expect(DeleteKamelet(ns, "config02-log-sink")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletConfigDefaultParamMountedSecret(t *testing.T) {
	RegisterTestingT(t)
	t.Run("run test default config secret properties", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "config03-timer-source")()).To(Succeed())
		Expect(CreateLogKamelet(ns, "config03-log-sink")()).To(Succeed())

		name := "config-test-timer-source-int3"
		secretName := "my-config-int3-secret"

		var secData = make(map[string]string)
		secData["my-message"] = "My secret message 03"
		secData["my-logger"] = "mySecretIntegrationLogger03"
		Expect(CreatePlainTextSecret(ns, secretName, secData)).To(Succeed())

		Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-parameters-configuration-03.yaml",
			"-t", "mount.configs=secret:"+secretName,
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My secret message 03"))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("mySecretIntegrationLogger03"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteSecret(ns, secretName)).To(Succeed())
		Expect(DeleteKamelet(ns, "config03-timer-source")).To(Succeed())
		Expect(DeleteKamelet(ns, "config03-log-sink")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestKameletConfigDefaultParamMountedConfigmap(t *testing.T) {
	RegisterTestingT(t)
	t.Run("run test default config configmap properties", func(t *testing.T) {

		Expect(CreateTimerKamelet(ns, "config04-timer-source")()).To(Succeed())
		Expect(CreateLogKamelet(ns, "config04-log-sink")()).To(Succeed())

		name := "config-test-timer-source-int4"
		cmName := "my-config-int4-configmap"

		var cmData = make(map[string]string)
		cmData["my-message"] = "My configmap message 04"
		cmData["my-logger"] = "myConfigmapIntegrationLogger04"
		Expect(CreatePlainTextConfigmap(ns, cmName, cmData)).To(Succeed())

		Expect(KamelRunWithID(operatorID, ns, "files/timer-kamelet-integration-parameters-configuration-04.yaml",
			"-t", "mount.configs=configmap:"+cmName,
			"--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("My configmap message 04"))
		Eventually(IntegrationLogs(ns, name)).Should(ContainSubstring("myConfigmapIntegrationLogger04"))

		Expect(Kamel("delete", name, "-n", ns).Execute()).To(Succeed())
		Eventually(Integration(ns, name), TestTimeoutLong).Should(BeNil())
		Expect(DeleteConfigmap(ns, cmName)).To(Succeed())
		Expect(DeleteKamelet(ns, "config04-timer-source")).To(Succeed())
		Expect(DeleteKamelet(ns, "config04-log-sink")).To(Succeed())
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
