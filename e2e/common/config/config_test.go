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

package resources

import (
	"io/ioutil"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/gzip"
)

func TestRunConfigExamples(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		// Properties

		t.Run("Simple property", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/property-route.groovy", "-p", "my.message=test-property").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("test-property"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/property-file-route.groovy", "--property", "file:./files/my.properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "property-file-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property precedence", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/property-file-route.groovy", "-p", "my.key.2=universe", "-p", "file:./files/my.properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "property-file-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello universe"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Configmap

		// Store a configmap on the cluster
		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my-configmap-content"
		NewPlainTextConfigmap(ns, "my-cm", cmData)

		// Store a configmap with multiple values
		var cmDataMulti = make(map[string]string)
		cmDataMulti["my-configmap-key"] = "should-not-see-it"
		cmDataMulti["my-configmap-key-2"] = "my-configmap-content-2"
		NewPlainTextConfigmap(ns, "my-cm-multi", cmDataMulti)

		t.Run("Config configmap", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-configmap-route.groovy", "--config", "configmap:my-cm").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-configmap-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "config-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Resource configmap", func(t *testing.T) {
			// We can reuse the configmap created previously

			Expect(Kamel("run", "-n", ns, "./files/resource-configmap-route.groovy", "--resource", "configmap:my-cm").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-configmap-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Resource configmap with destination", func(t *testing.T) {
			// We can reuse the configmap created previously

			Expect(Kamel("run", "-n", ns, "./files/resource-configmap-location-route.groovy", "--resource", "configmap:my-cm@/tmp/app").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-configmap-location-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-configmap-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-configmap-location-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Resource configmap with filtered key and destination", func(t *testing.T) {
			// We'll use the configmap contaning 2 values filtering only 1 key

			Expect(Kamel("run", "-n", ns, "./files/resource-configmap-key-location-route.groovy", "--resource", "configmap:my-cm-multi/my-configmap-key-2@/tmp/app").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-configmap-key-location-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-configmap-key-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-configmap-key-location-route"), TestTimeoutShort).ShouldNot(ContainSubstring(cmDataMulti["my-configmap-key"]))
			Eventually(IntegrationLogs(ns, "resource-configmap-key-location-route"), TestTimeoutShort).Should(ContainSubstring(cmDataMulti["my-configmap-key-2"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Store a configmap as property file
		var cmDataProps = make(map[string]string)
		cmDataProps["my.properties"] = "my.key.1=hello\nmy.key.2=world"
		NewPlainTextConfigmap(ns, "my-cm-properties", cmDataProps)

		t.Run("Config configmap as property file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-configmap-properties-route.groovy", "--config", "configmap:my-cm-properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-configmap-properties-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "config-configmap-properties-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-configmap-properties-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Secret

		// Store a secret on the cluster
		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret"
		NewPlainTextSecret(ns, "my-sec", secData)

		t.Run("Config secret", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-secret-route.groovy", "--config", "secret:my-sec").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-secret-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "config-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Resource secret", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-secret-route.groovy", "--resource", "secret:my-sec").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-secret-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Config File

		t.Run("Plain text configuration file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-file-route.groovy", "--config", "file:./files/resources-data.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-file-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "config-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(AutogeneratedConfigmapsCount(ns), TestTimeoutShort).Should(Equal(0))
		})

		// Resource File

		t.Run("Plain text resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-file-route.groovy", "--resource", "file:./files/resources-data.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(AutogeneratedConfigmapsCount(ns), TestTimeoutShort).Should(Equal(0))
		})

		t.Run("Plain text resource file with destination path", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-file-location-route.groovy", "--resource", "file:./files/resources-data.txt@/tmp/file.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-location-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-file-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-location-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(AutogeneratedConfigmapsCount(ns), TestTimeoutShort).Should(Equal(0))
		})

		t.Run("Binary (zip) resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-file-binary-route.groovy", "--resource", "file:./files/resources-data.zip", "-d", "camel-zipfile").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-binary-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-file-binary-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-binary-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(AutogeneratedConfigmapsCount(ns), TestTimeoutShort).Should(Equal(0))
		})

		t.Run("Base64 compressed binary resource file", func(t *testing.T) {
			// We calculate the expected content
			source, err := ioutil.ReadFile("./files/resources-data.txt")
			assert.Nil(t, err)
			expectedBytes, err := gzip.CompressBase64([]byte(source))
			assert.Nil(t, err)

			Expect(Kamel("run", "-n", ns, "./files/resource-file-base64-encoded-route.groovy", "--resource", "file:./files/resources-data.txt", "--compression=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-base64-encoded-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "resource-file-base64-encoded-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-base64-encoded-route"), TestTimeoutShort).Should(ContainSubstring(string(expectedBytes)))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(AutogeneratedConfigmapsCount(ns), TestTimeoutShort).Should(Equal(0))
		})

		// Build-Properties
		t.Run("Build time property", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-route.groovy", "--build-property", "quarkus.application.name=my-super-application").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "build-property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			// Don't delete - we need it for next test execution
		})

		// We need to check also that the property (which is available in the IntegrationKit) is correctly replaced and we don't reuse the same kit
		t.Run("Build time property updated", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-route.groovy", "--name", "build-property-route-updated",
				"--build-property", "quarkus.application.name=my-super-application-updated").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-route-updated"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "build-property-route-updated", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-route-updated"), TestTimeoutShort).Should(ContainSubstring("my-super-application-updated"))
			// Verify the integration kits are different
			Expect(IntegrationKit(ns, "build-property-route")).ShouldNot(Equal(IntegrationKit(ns, "build-property-route-updated")))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Build-Properties file
		t.Run("Build time property file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-file-route.groovy", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Build time property file with precedence", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-file-route.groovy", "--build-property", "quarkus.application.name=my-overridden-application", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutMedium).Should(ContainSubstring("my-overridden-application"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

	})
}
