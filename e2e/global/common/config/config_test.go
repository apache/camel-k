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
	"fmt"
	"io/ioutil"
	"testing"

	"github.com/stretchr/testify/assert"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/gzip"
)

func operatorID(ns string) string {
	return fmt.Sprintf("camel-k-%s", ns)
}

func installWithID(ns string) {
	Expect(KamelInstallWithID(operatorID(ns), ns).Execute()).To(Succeed())
}

func TestRunConfigExampleSimpleProperty(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		// Properties
		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-route.groovy", "-p", "my.message=test-property").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("test-property"))
	})
}

func TestRunConfigExamplePropertyFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-file-route.groovy", "--property", "file:./files/my.properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
	})
}

func TestRunConfigExamplePropertyPrecedence(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-file-route.groovy", "-p", "my.key.2=universe", "-p", "file:./files/my.properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello universe"))
	})
}

func TestRunConfigExamplePropertyConfigMap(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var cmData = make(map[string]string)
		cmData["my.message"] = "my-configmap-property-value"
		CreatePlainTextConfigmap(ns, "my-cm-test-property", cmData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-route.groovy", "-p", "configmap:my-cm-test-property").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-configmap-property-value"))
	})
}

func TestRunConfigExampleConfigMapPropertyFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var cmData = make(map[string]string)
		cmData["my.properties"] = "my.message=my-configmap-property-entry"
		CreatePlainTextConfigmap(ns, "my-cm-test-properties", cmData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-route.groovy", "-p", "configmap:my-cm-test-properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-configmap-property-entry"))
	})
}

func TestRunConfigExamplePropertySecret(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var secData = make(map[string]string)
		secData["my.message"] = "my-secret-property-value"
		CreatePlainTextSecret(ns, "my-sec-test-property", secData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-route.groovy", "-p", "secret:my-sec-test-property").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-secret-property-value"))
	})
}

func TestRunConfigExampleSecretPropertyFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var secData = make(map[string]string)
		secData["my.properties"] = "my.message=my-secret-property-entry"
		CreatePlainTextSecret(ns, "my-sec-test-properties", secData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/property-route.groovy", "-p", "secret:my-sec-test-properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-secret-property-entry"))
	})
}

// Store a configmap on the cluster
func createConfigMap(ns string) map[string]string {
	var cmData = make(map[string]string)
	cmData["my-configmap-key"] = "my-configmap-content"
	CreatePlainTextConfigmap(ns, "my-cm", cmData)
	return cmData
}

// Store a configmap with multiple values
func createMultiConfigMap(ns string) map[string]string {
	var cmDataMulti = make(map[string]string)
	cmDataMulti["my-configmap-key"] = "should-not-see-it"
	cmDataMulti["my-configmap-key-2"] = "my-configmap-content-2"
	CreatePlainTextConfigmap(ns, "my-cm-multi", cmDataMulti)
	return cmDataMulti
}

func TestRunConfigPropertyConfigMap(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		cmData := createConfigMap(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/config-configmap-route.groovy", "--config", "configmap:my-cm").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "config-configmap-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "config-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "config-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
	})
}

func TestRunConfigResourceConfigMap(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		cmData := createConfigMap(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-configmap-route.groovy", "--resource", "configmap:my-cm").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-configmap-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
	})
}

func TestRunConfigDestinationConfigMap(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		cmData := createConfigMap(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-configmap-location-route.groovy", "--resource", "configmap:my-cm@/tmp/app").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-configmap-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-configmap-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-configmap-location-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
	})
}

func TestRunConfigFilteredConfigMap(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		cmDataMulti := createMultiConfigMap(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-configmap-key-location-route.groovy", "--resource", "configmap:my-cm-multi/my-configmap-key-2@/tmp/app/test.txt").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-configmap-key-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-configmap-key-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-configmap-key-location-route"), TestTimeoutShort).ShouldNot(ContainSubstring(cmDataMulti["my-configmap-key"]))
		Eventually(IntegrationLogs(ns, "resource-configmap-key-location-route"), TestTimeoutShort).Should(ContainSubstring(cmDataMulti["my-configmap-key-2"]))
	})
}

func TestRunConfigPropertyConfigMapFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		// Store a configmap as property file
		var cmDataProps = make(map[string]string)
		cmDataProps["my.properties"] = "my.key.1=hello\nmy.key.2=world"
		CreatePlainTextConfigmap(ns, "my-cm-properties", cmDataProps)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/config-configmap-properties-route.groovy", "--config", "configmap:my-cm-properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "config-configmap-properties-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "config-configmap-properties-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "config-configmap-properties-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
	})
}

func createSecret(ns string) map[string]string {
	// Store a secret on the cluster
	var secData = make(map[string]string)
	secData["my-secret-key"] = "very top secret"
	CreatePlainTextSecret(ns, "my-sec", secData)
	return secData
}

func createMultiSecret(ns string) map[string]string {
	// Store a secret with multi values
	var secDataMulti = make(map[string]string)
	secDataMulti["my-secret-key"] = "very top secret"
	secDataMulti["my-secret-key-2"] = "even more secret"
	CreatePlainTextSecret(ns, "my-sec-multi", secDataMulti)
	return secDataMulti
}

func TestRunConfigPropertyConfigSecret(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		secData := createSecret(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/config-secret-route.groovy", "--config", "secret:my-sec").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "config-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "config-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "config-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
	})
}

func TestRunConfigPropertyResourceSecret(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		secData := createSecret(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-secret-route.groovy", "--resource", "secret:my-sec").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
	})
}

func TestRunConfigSecretFilteredKey(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)
		secDataMulti := createMultiSecret(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/config-secret-key-route.groovy", "--config", "secret:my-sec-multi/my-secret-key-2").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "config-secret-key-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "config-secret-key-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "config-secret-key-route"), TestTimeoutShort).ShouldNot(ContainSubstring(secDataMulti["my-secret-key"]))
		Eventually(IntegrationLogs(ns, "config-secret-key-route"), TestTimeoutShort).Should(ContainSubstring(secDataMulti["my-secret-key-2"]))
	})
}

// Config File
func TestRunConfigPlainTextFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/config-file-route.groovy", "--config", "file:./files/resources-data.txt").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "config-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "config-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "config-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
	})
}

// Resource File
func TestRunConfigResourceFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-file-route.groovy", "--resource", "file:./files/resources-data.txt").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
	})
}

func TestRunConfigDestinationPlainTextFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-file-location-route.groovy", "--resource", "file:./files/resources-data.txt@/tmp/file.txt").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-file-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-file-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-file-location-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
	})
}

func TestRunConfigZipFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-file-binary-route.groovy", "--resource", "file:./files/resources-data.zip", "-d", "camel:zipfile").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-file-binary-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-file-binary-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-file-binary-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
	})
}

func TestRunConfigBase64File(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		// We calculate the expected content
		source, err := ioutil.ReadFile("./files/resources-data.txt")
		assert.Nil(t, err)
		expectedBytes, err := gzip.CompressBase64([]byte(source))
		assert.Nil(t, err)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-file-base64-encoded-route.groovy", "--resource", "file:./files/resources-data.txt", "--compression=true").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-file-base64-encoded-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-file-base64-encoded-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-file-base64-encoded-route"), TestTimeoutShort).Should(ContainSubstring(string(expectedBytes)))
	})
}

func TestRunConfigPlainTextFileSameContent(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/resource-file-route.groovy", "--resource", "file:./files/resources-data.txt",
			"--resource", "file:./files/resources-data-same.txt").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "resource-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "resource-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "resource-file-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
	})
}

func TestRunConfigBuildProperty(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		// Build-Properties
		t.Run("Build time property", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-route.groovy", "--build-property", "quarkus.application.name=my-super-application").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "build-property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			// Don't delete - we need it for next test execution
		})

		// We need to check also that the property (which is available in the IntegrationKit) is correctly replaced and we don't reuse the same kit
		t.Run("Build time property updated", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-route.groovy", "--name", "build-property-route-updated",
				"--build-property", "quarkus.application.name=my-super-application-updated").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "build-property-route-updated"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "build-property-route-updated", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "build-property-route-updated"), TestTimeoutShort).Should(ContainSubstring("my-super-application-updated"))
			// Verify the integration kits are different
			Expect(IntegrationKit(ns, "build-property-route")).ShouldNot(Equal(IntegrationKit(ns, "build-property-route-updated")))
		})
	})
}

func TestRunConfigBuildPropertyFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		// Build-Properties file
		Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-file-route.groovy", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
	})
}

func TestRunConfigBuildPropertytFilePrecedence(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-file-route.groovy", "--build-property", "quarkus.application.name=my-overridden-application", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutMedium).Should(ContainSubstring("my-overridden-application"))
	})
}

func TestRunConfigBuildPropertyConfigMap(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var cmData = make(map[string]string)
		cmData["quarkus.application.name"] = "my-cool-application"
		CreatePlainTextConfigmap(ns, "my-cm-test-build-property", cmData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-file-route.groovy", "--build-property", "configmap:my-cm-test-build-property").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutLong).Should(ContainSubstring("my-cool-application"))
	})
}

func TestRunConfigBuildPropertyConfigMapFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var cmData = make(map[string]string)
		cmData["my.properties"] = "quarkus.application.name=my-super-cool-application"
		CreatePlainTextConfigmap(ns, "my-cm-test-build-properties", cmData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-file-route.groovy", "--build-property", "configmap:my-cm-test-build-properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-super-cool-application"))
	})
}

func TestRunConfigBuildPropertySecret(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var secData = make(map[string]string)
		secData["quarkus.application.name"] = "my-great-application"
		CreatePlainTextSecret(ns, "my-sec-test-build-property", secData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-file-route.groovy", "--build-property", "secret:my-sec-test-build-property").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-great-application"))
	})
}

func TestRunConfigBuildPropertySecretFile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		var secData = make(map[string]string)
		secData["my.properties"] = "quarkus.application.name=my-awsome-application"
		CreatePlainTextSecret(ns, "my-sec-test-build-properties", secData)

		Expect(KamelRunWithID(operatorID(ns), ns, "./files/build-property-file-route.groovy", "--build-property", "secret:my-sec-test-build-properties").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-awsome-application"))
	})
}
