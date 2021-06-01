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

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/gzip"
)

func TestRunConfigExamples(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		// Properties

		t.Run("Simple property", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/property-route.groovy", "-p", "my.message=test-property").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "property-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("test-property"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/property-file-route.groovy", "--property", "file:./files/my.properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "property-file-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "property-file-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Configmap

		// Store a configmap on the cluster
		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my-configmap-content"
		NewPlainTextConfigmap(ns, "my-cm", cmData)

		t.Run("Config configmap", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-configmap-route.groovy", "--config", "configmap:my-cm").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-configmap-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "config-configmap-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Resource configmap", func(t *testing.T) {
			// We can reuse the configmap created previously

			Expect(Kamel("run", "-n", ns, "./files/resource-configmap-route.groovy", "--resource", "configmap:my-cm").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-configmap-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resource-configmap-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Secret

		// Store a secret on the cluster
		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret"
		NewPlainTextSecret(ns, "my-sec", secData)

		t.Run("Config secret", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-secret-route.groovy", "--config", "secret:my-sec").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-secret-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "config-secret-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Resource secret", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-secret-route.groovy", "--resource", "secret:my-sec").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-secret-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resource-secret-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Config File

		t.Run("Plain text configuration file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/config-file-route.groovy", "--config", "file:./files/resources-data.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "config-file-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "config-file-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "config-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Resource File

		t.Run("Plain text resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-file-route.groovy", "--resource", "file:./files/resources-data.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resource-file-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Binary (zip) resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resource-file-binary-route.groovy", "--resource", "file:./files/resources-data.zip", "-d", "camel-zipfile").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-binary-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resource-file-binary-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-binary-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Base64 compressed binary resource file", func(t *testing.T) {
			// We calculate the expected content
			source, err := ioutil.ReadFile("./files/resources-data.txt")
			assert.Nil(t, err)
			expectedBytes, err := gzip.CompressBase64([]byte(source))
			assert.Nil(t, err)

			Expect(Kamel("run", "-n", ns, "./files/resource-file-base64-encoded-route.groovy", "--resource", "file:./files/resources-data.txt", "--compression=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resource-file-base64-encoded-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resource-file-base64-encoded-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resource-file-base64-encoded-route"), TestTimeoutShort).Should(ContainSubstring(string(expectedBytes)))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Build-Properties
		t.Run("Build time property", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-route.groovy", "--build-property", "quarkus.application.name=my-super-application").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "build-property-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			// Don't delete - we need it for next test execution
		})

		// We need to check also that the property (which is available in the IntegrationKit) is correctly replaced and we don't reuse the same kit
		t.Run("Build time property updated", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-route.groovy", "--name", "build-property-route-updated",
				"--build-property", "quarkus.application.name=my-super-application-updated").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-route-updated"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "build-property-route-updated", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-route-updated"), TestTimeoutShort).Should(ContainSubstring("my-super-application-updated"))
			// Verify the integration kits are different
			Expect(IntegrationKit(ns, "build-property-route")).ShouldNot(Equal(IntegrationKit(ns, "build-property-route-updated")))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Build-Properties file
		t.Run("Build time property file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/build-property-file-route.groovy", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "build-property-file-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
