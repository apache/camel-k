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

		t.Run("Textplain configmap", func(t *testing.T) {
			// Store a configmap on the cluster
			var cmData = make(map[string]string)
			cmData["my-configmap-key"] = "my-configmap-content"
			NewPlainTextConfigmap(ns, "my-cm", cmData)

			Expect(Kamel("run", "-n", ns, "./files/configmap-route.groovy", "--configmap", "my-cm").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "configmap-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "configmap-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Secret

		t.Run("Textplain secret", func(t *testing.T) {
			// Store a secret on the cluster
			var secData = make(map[string]string)
			secData["my-secret-key"] = "very top secret"
			NewPlainTextSecret(ns, "my-sec", secData)

			Expect(Kamel("run", "-n", ns, "./files/secret-route.groovy", "--secret", "my-sec").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "secret-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "secret-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Resources

		t.Run("Plain text resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resources-route.groovy", "--resource", "./files/resources-data.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resources-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Binary (zip) resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resources-binary-route.groovy", "--resource", "./files/resources-data.zip", "-d", "camel-zipfile").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resources-binary-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-binary-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-binary-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Base64 compressed binary resource file", func(t *testing.T) {
			// We calculate the expected content
			source, err := ioutil.ReadFile("./files/resources-data.txt")
			assert.Nil(t, err)
			expectedBytes, err := gzip.CompressBase64([]byte(source))
			assert.Nil(t, err)

			Expect(Kamel("run", "-n", ns, "./files/resources-base64-encoded-route.groovy", "--resource", "./files/resources-data.txt", "--compression=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resources-base64-encoded-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-base64-encoded-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-base64-encoded-route"), TestTimeoutShort).Should(ContainSubstring(string(expectedBytes)))
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
