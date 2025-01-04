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

package common

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestRunConfigProperties(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Simple property", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/property-route.yaml", "-p", "my.message=test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("test-property"))
		})

		t.Run("Property file", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/property-file-route.yaml", "--property", "file:./files/my.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello world"))
		})

		t.Run("Property precedence", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/property-file-route.yaml", "-p", "my.key.2=universe", "-p", "file:./files/my.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-file-route"), TestTimeoutShort).Should(ContainSubstring("hello universe"))
		})

		t.Run("Property from ConfigMap", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["my.message"] = "my-configmap-property-value"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-property", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/property-route.yaml", "-p", "configmap:my-cm-test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-configmap-property-value"))
		})

		t.Run("Property from Secret", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["my.message"] = "my-secret-property-value"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-property", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/property-route.yaml", "-p", "secret:my-sec-test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "property-route"), TestTimeoutShort).Should(ContainSubstring("my-secret-property-value"))
		})

	})
}

func TestRunConfigConfigmaps(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Store a configmap on the cluster
		// kubectl create configmap my-cm --from-literal=my-configmap-key="my-configmap-content"

		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my-configmap-content"
		err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm", cmData)
		g.Expect(err).To(BeNil())

		// Store a configmap with multiple values
		// kubectl create configmap my-cm-multi --from-literal=my-configmap-key="should-not-see-it" --from-literal=my-configmap-key-2="my-configmap-content-2"

		var cmDataMulti = make(map[string]string)
		cmDataMulti["my-configmap-key"] = "should-not-see-it"
		cmDataMulti["my-configmap-key-2"] = "my-configmap-content-2"
		err = CreatePlainTextConfigmap(t, ctx, ns, "my-cm-multi", cmDataMulti)
		g.Expect(err).To(BeNil())

		t.Run("Config configmap", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-route.yaml", "--config", "configmap:my-cm").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-configmap-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap", func(t *testing.T) {
			// We can reuse the configmap created previously
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-configmap-route.yaml", "--resource", "configmap:my-cm").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-configmap-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-configmap-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap with destination", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-configmap-location-route.yaml", "--resource", "configmap:my-cm@/tmp/app").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-configmap-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-configmap-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-location-route"), TestTimeoutShort).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap with filtered key and destination", func(t *testing.T) {
			// We'll use the configmap containing 2 values filtering only 1 key
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-configmap-key-location-route.yaml", "--resource", "configmap:my-cm-multi/my-configmap-key-2@/tmp/app/test.txt").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-configmap-key-location-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-configmap-key-location-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-key-location-route"), TestTimeoutShort).ShouldNot(ContainSubstring(cmDataMulti["my-configmap-key"]))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-configmap-key-location-route"), TestTimeoutShort).Should(ContainSubstring(cmDataMulti["my-configmap-key-2"]))
		})
	})
}

func TestRunConfigSecrets(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Store a secret on the cluster
		// kubectl create secret generic my-sec --from-literal=my-secret-key="very top secret"

		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret"
		err := CreatePlainTextSecret(t, ctx, ns, "my-sec", secData)
		g.Expect(err).To(BeNil())

		// Store a secret with multi values
		// kubectl create secret generic my-sec-multi --from-literal=my-secret-key="very top secret" --from-literal=my-secret-key-2="even more secret"

		var secDataMulti = make(map[string]string)
		secDataMulti["my-secret-key"] = "very top secret"
		secDataMulti["my-secret-key-2"] = "even more secret"
		err = CreatePlainTextSecret(t, ctx, ns, "my-sec-multi", secDataMulti)
		g.Expect(err).To(BeNil())

		t.Run("Config secret", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/config-secret-route.yaml", "--config", "secret:my-sec").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
		})

		t.Run("Resource secret", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-secret-route.yaml", "--resource", "secret:my-sec").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "resource-secret-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "resource-secret-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "resource-secret-route"), TestTimeoutShort).Should(ContainSubstring(secData["my-secret-key"]))
		})

		/*
			kamel run --dev --config secret:my-sec-multi/my-secret-key-2 ./e2e/common/config/files/config-secret-key-route.yaml
		*/

		t.Run("Secret with filtered key", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/config-secret-key-route.yaml", "--config", "secret:my-sec-multi/my-secret-key-2").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "config-secret-key-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "config-secret-key-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-secret-key-route"), TestTimeoutShort).ShouldNot(ContainSubstring(secDataMulti["my-secret-key"]))
			g.Eventually(IntegrationLogs(t, ctx, ns, "config-secret-key-route"), TestTimeoutShort).Should(ContainSubstring(secDataMulti["my-secret-key-2"]))
		})

	})
}

func TestRunConfigBuildProperties(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Build time property", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-route.yaml", "--build-property", "quarkus.application.name=my-super-application").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
			// Don't delete - we need it for next test execution
		})

		// We need to check also that the property (which is available in the IntegrationKit) is correctly replaced and that we don't reuse the same kit
		t.Run("Build time property updated", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-route.yaml", "--name", "build-property-route-updated", "--build-property", "quarkus.application.name=my-super-application-updated").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-route-updated"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-route-updated", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-route-updated"), TestTimeoutShort).Should(ContainSubstring("my-super-application-updated"))
			// Verify the integration kits are different
			g.Eventually(IntegrationKit(t, ctx, ns, "build-property-route-updated")).ShouldNot(Equal(IntegrationKit(t, ctx, ns, "build-property-route")()))
		})

		// Build-Properties file
		t.Run("Build time property file", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-super-application"))
		})

		t.Run("Build time property file with precedence", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml", "--name", "build-property-file-route-precedence", "--build-property", "quarkus.application.name=my-overridden-application", "--build-property", "file:./files/quarkus.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route-precedence"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route-precedence", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route-precedence"), TestTimeoutShort).Should(ContainSubstring("my-overridden-application"))
		})

		t.Run("Build time property from ConfigMap", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["quarkus.application.name"] = "my-cool-application"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-build-property", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml", "--build-property", "configmap:my-cm-test-build-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-cool-application"))
		})

		t.Run("Build time property from ConfigMap as property file", func(t *testing.T) {
			var cmData = make(map[string]string)
			cmData["my.properties"] = "quarkus.application.name=my-super-cool-application"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-build-properties", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml", "--name", "build-property-file-route-cm", "--build-property", "configmap:my-cm-test-build-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route-cm"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route-cm", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route-cm"), TestTimeoutShort).Should(ContainSubstring("my-super-cool-application"))
		})

		t.Run("Build time property from Secret", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["quarkus.application.name"] = "my-great-application"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-build-property", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml", "--build-property", "secret:my-sec-test-build-property").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route"), TestTimeoutShort).Should(ContainSubstring("my-great-application"))
		})

		t.Run("Build time property from Secret as property file", func(t *testing.T) {
			var secData = make(map[string]string)
			secData["my.properties"] = "quarkus.application.name=my-awesome-application"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-build-properties", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml", "--name", "build-property-file-route-secret", "--build-property", "secret:my-sec-test-build-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "build-property-file-route-secret"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "build-property-file-route-secret", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "build-property-file-route-secret"), TestTimeoutShort).Should(ContainSubstring("my-awesome-application"))
		})

	})
}
