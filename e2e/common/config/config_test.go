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
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestRunConfigExamples(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-config"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("Simple property", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-route.groovy", "-p", "my.message=test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("test-property"))
			g.Expect(Kamel(t, ctx, "delete", "property-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property file", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-file-route.groovy", "--property", "file:./files/my.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
			g.Expect(Kamel(t, ctx, "delete", "property-file-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property precedence", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-file-route.groovy", "-p", "my.key.2=universe", "-p", "file:./files/my.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello universe"))
			g.Expect(Kamel(t, ctx, "delete", "property-file-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property from ConfigMap", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["my.message"] = "my-configmap-property-value"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-property", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-route.groovy", "-p", "configmap:my-cm-test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-configmap-property-value"))
			g.Expect(Kamel(t, ctx, "delete", "property-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property from ConfigMap as property file", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["my.properties"] = "my.message=my-configmap-property-entry"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-properties", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-route.groovy", "-p", "configmap:my-cm-test-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-configmap-property-entry"))
			g.Expect(Kamel(t, ctx, "delete", "property-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property from Secret", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["my.message"] = "my-secret-property-value"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-property", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-route.groovy", "-p", "secret:my-sec-test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-secret-property-value"))
			g.Expect(Kamel(t, ctx, "delete", "property-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Property from Secret as property file", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["my.properties"] = "my.message=my-secret-property-entry"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-properties", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-route.groovy", "--name", "property-route-secret", "-p", "secret:my-sec-test-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route-secret"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route-secret", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route-secret"), TestTimeoutShort).Should(ContainSubstring("my-secret-property-entry"))
		})

		t.Run("Property from Secret inlined", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["my-message"] = "my-secret-external-value"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-inlined", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/property-secret-route.groovy", "-t", "mount.configs=secret:my-sec-inlined").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-secret-route"), TestTimeoutShort).Should(ContainSubstring("my-secret-external-value"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, "property-secret-route")).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, "property-secret-route")()
			mountTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "mount")
			g.Expect(mountTrait).ToNot(BeNil())
			g.Expect(len(mountTrait)).To(Equal(1))
			g.Expect(mountTrait["configs"]).ToNot(BeNil())

			g.Expect(Kamel(t, ctx, "delete", "property-secret-route", "-n", ns).Execute()).To(Succeed())

		})

		// Configmap

		// Store a configmap on the cluster
		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my-configmap-content"
		err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm", cmData)
		g.Expect(err).To(BeNil())

		// Store a configmap with multiple values
		var cmDataMulti = make(map[string]string)
		cmDataMulti["my-configmap-key"] = "should-not-see-it"
		cmDataMulti["my-configmap-key-2"] = "my-configmap-content-2"
		err = CreatePlainTextConfigmap(t, ctx, ns, "my-cm-multi", cmDataMulti)
		g.Expect(err).To(BeNil())

		t.Run("Config configmap", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/config-configmap-route.groovy", "--config", "configmap:my-cm").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-configmap-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap", func(t *testing.T) {
			// We can reuse the configmap created previously

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/resource-configmap-route.groovy", "--resource", "configmap:my-cm").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-configmap-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap with destination", func(t *testing.T) {
			// We can reuse the configmap created previously

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/resource-configmap-location-route.groovy", "--resource", "configmap:my-cm@/tmp/app").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-configmap-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-configmap-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-location-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap with filtered key and destination", func(t *testing.T) {
			// We'll use the configmap contaning 2 values filtering only 1 key

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/resource-configmap-key-location-route.groovy", "--resource", "configmap:my-cm-multi/my-configmap-key-2@/tmp/app/test.txt").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-configmap-key-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-configmap-key-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-key-location-route"), TestTimeoutShort).ShouldNot(ContainSubstring(cmDataMulti["my-configmap-key"]))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-key-location-route"), TestTimeoutShort).Should(ContainSubstring(cmDataMulti["my-configmap-key-2"]))
		})

		t.Run("Config configmap as property file", func(t *testing.T) {
			// Store a configmap as property file
			var cmDataProps = make(map[string]string)
			cmDataProps["my.properties"] = "my.key.1=hello\nmy.key.2=world"
			err = CreatePlainTextConfigmap(t, ctx, ns, "my-cm-properties", cmDataProps)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/config-configmap-properties-route.groovy", "--config", "configmap:my-cm-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-configmap-properties-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-configmap-properties-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-configmap-properties-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
		})

		// Secret

		// Store a secret on the cluster
		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret"
		err = CreatePlainTextSecret(t, ctx, ns, "my-sec", secData)
		g.Expect(err).To(BeNil())

		// Store a secret with multi values
		var secDataMulti = make(map[string]string)
		secDataMulti["my-secret-key"] = "very top secret"
		secDataMulti["my-secret-key-2"] = "even more secret"
		err = CreatePlainTextSecret(t, ctx, ns, "my-sec-multi", secDataMulti)
		g.Expect(err).To(BeNil())

		t.Run("Config secret", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/config-secret-route.groovy", "--config", "secret:my-sec").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
		})

		t.Run("Resource secret", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/resource-secret-route.groovy", "--resource", "secret:my-sec").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Secret with filtered key", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/config-secret-key-route.groovy", "--config", "secret:my-sec-multi/my-secret-key-2").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-secret-key-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-secret-key-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-secret-key-route"), TestTimeoutShort).ShouldNot(ContainSubstring(secDataMulti["my-secret-key"]))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-secret-key-route"), TestTimeoutShort).Should(ContainSubstring(secDataMulti["my-secret-key-2"]))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Build-Properties
		t.Run("Build time property", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-route.groovy", "--build-property", "quarkus.application.name=my-super-application").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			// Don't delete - we need it for next test execution
		})

		// We need to check also that the property (which is available in the IntegrationKit) is correctly replaced and we don't reuse the same kit
		t.Run("Build time property updated", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-route.groovy", "--name", "build-property-route-updated", "--build-property", "quarkus.application.name=my-super-application-updated").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-route-updated"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-route-updated", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-route-updated"), TestTimeoutShort).Should(ContainSubstring("my-super-application-updated"))
			// Verify the integration kits are different
			g.Eventually(IntegrationKit(t, ctx, ns, "build-property-route-updated")).ShouldNot(Equal(IntegrationKit(t, ctx, ns, "build-property-route")()))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Build-Properties file
		t.Run("Build time property file", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-file-route.groovy", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			g.Expect(Kamel(t, ctx, "delete", "build-property-file-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Build time property file with precedence", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-file-route.groovy", "--name", "build-property-file-route-precedence", "--build-property", "quarkus.application.name=my-overridden-application", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route-precedence"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route-precedence", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route-precedence"), TestTimeoutShort).Should(ContainSubstring("my-overridden-application"))
			g.Expect(Kamel(t, ctx, "delete", "build-property-file-route-precedence", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Build time property from ConfigMap", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["quarkus.application.name"] = "my-cool-application"
			err = CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-build-property", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-file-route.groovy", "--build-property", "configmap:my-cm-test-build-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-cool-application"))
			g.Expect(Kamel(t, ctx, "delete", "build-property-file-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Build time property from ConfigMap as property file", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["my.properties"] = "quarkus.application.name=my-super-cool-application"
			err = CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-build-properties", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-file-route.groovy", "--name", "build-property-file-route-cm", "--build-property", "configmap:my-cm-test-build-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route-cm"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route-cm", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route-cm"), TestTimeoutShort).Should(ContainSubstring("my-super-cool-application"))
			g.Expect(Kamel(t, ctx, "delete", "build-property-file-route-cm", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Build time property from Secret", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["quarkus.application.name"] = "my-great-application"
			err = CreatePlainTextSecret(t, ctx, ns, "my-sec-test-build-property", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-file-route.groovy", "--build-property", "secret:my-sec-test-build-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-great-application"))
			g.Expect(Kamel(t, ctx, "delete", "build-property-file-route", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Build time property from Secret as property file", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["my.properties"] = "quarkus.application.name=my-awsome-application"
			err = CreatePlainTextSecret(t, ctx, ns, "my-sec-test-build-properties", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "./files/build-property-file-route.groovy", "--name", "build-property-file-route-secret", "--build-property", "secret:my-sec-test-build-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route-secret"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route-secret", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route-secret"), TestTimeoutShort).Should(ContainSubstring("my-awsome-application"))
			g.Expect(Kamel(t, ctx, "delete", "build-property-file-route-secret", "-n", ns).Execute()).To(Succeed())
		})

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
