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
			name := RandomizedSuffixName("property-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/property-route.yaml",
				"--name", name,
				"-p", "my.message=test-property",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("test-property"))
		})

		t.Run("Property file", func(t *testing.T) {
			name := RandomizedSuffixName("property-file-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/property-file-route.yaml",
				"--name", name,
				"--property", "file:./files/my.properties").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("hello world"))
		})

		t.Run("Property precedence", func(t *testing.T) {
			name := RandomizedSuffixName("property-file-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/property-file-route.yaml",
				"--name", name,
				"-p", "my.key.2=universe",
				"-p", "file:./files/my.properties",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("hello universe"))
		})

		t.Run("Property from ConfigMap", func(t *testing.T) {
			name := RandomizedSuffixName("property-route")
			var cmData = make(map[string]string)
			cmData["my.message"] = "my-configmap-property-value"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-property", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/property-route.yaml",
				"--name", name,
				"-p", "configmap:my-cm-test-property",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-configmap-property-value"))
		})

		t.Run("Property from Secret", func(t *testing.T) {
			name := RandomizedSuffixName("property-route")
			var secData = make(map[string]string)
			secData["my.message"] = "my-secret-property-value"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-property", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/property-route.yaml",
				"--name", name,
				"-p", "secret:my-sec-test-property").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-secret-property-value"))
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

		// Store a configmap that mocks the '--from-file' functionality
		// kubectl create configmap my-cm-properties-file --from-file=./files/my.properties"

		err = CreateFromFileConfigmap(t, ctx, ns, "my-cm-properties-file", "./files/my.properties")
		g.Expect(err).To(BeNil())

		t.Run("Config configmap", func(t *testing.T) {
			name := RandomizedSuffixName("config-configmap-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-route.yaml",
				"--name", name,
				"--config", "configmap:my-cm",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Config configmap from properties file", func(t *testing.T) {
			name := RandomizedSuffixName("config-configmap-properties-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-properties-route.yaml",
				"--name", name,
				"--config", "configmap:my-cm-properties-file",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("hello world"))
		})

		t.Run("Config configmap from properties file (interpolated)", func(t *testing.T) {
			name := RandomizedSuffixName("config-configmap-properties-interpolation-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-properties-interpolation-route.yaml",
				"--name", name,
				"--config", "configmap:my-cm-properties-file").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("hello world"))
		})

		t.Run("Resource configmap", func(t *testing.T) {
			name := RandomizedSuffixName("resource-configmap-route")
			// We can reuse the configmap created previously
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-configmap-route.yaml",
				"--name", name,
				"--resource", "configmap:my-cm").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap with destination", func(t *testing.T) {
			name := RandomizedSuffixName("resource-configmap-location-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-configmap-location-route.yaml",
				"--name", name,
				"--resource", "configmap:my-cm@/tmp/app",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(cmData["my-configmap-key"]))
		})

		t.Run("Resource configmap with filtered key and destination", func(t *testing.T) {
			name := RandomizedSuffixName("resource-configmap-key-location-route")
			// We'll use the configmap containing 2 values filtering only 1 key
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-configmap-key-location-route.yaml",
				"--name", name,
				"--resource", "configmap:my-cm-multi/my-configmap-key-2@/tmp/app/test.txt",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).ShouldNot(ContainSubstring(cmDataMulti["my-configmap-key"]))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(cmDataMulti["my-configmap-key-2"]))
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
		// kubectl create secret generic my-sec-multi --from-literal=my-secret-key="very top secret"
		// --from-literal=my-secret-key-2="even more secret"

		var secDataMulti = make(map[string]string)
		secDataMulti["my-secret-key"] = "very top secret"
		secDataMulti["my-secret-key-2"] = "even more secret"
		err = CreatePlainTextSecret(t, ctx, ns, "my-sec-multi", secDataMulti)
		g.Expect(err).To(BeNil())

		t.Run("Config secret", func(t *testing.T) {
			name := RandomizedSuffixName("config-secret-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/config-secret-route.yaml",
				"--name", name,
				"--config", "secret:my-sec",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(secData["my-secret-key"]))
		})

		t.Run("Resource secret", func(t *testing.T) {
			name := RandomizedSuffixName("resource-secret-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/resource-secret-route.yaml",
				"--name", name,
				"--resource", "secret:my-sec",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(secData["my-secret-key"]))
		})

		/*
			kamel run --config secret:my-sec-multi/my-secret-key-2 ./e2e/common/config/files/config-secret-key-route.yaml
		*/

		t.Run("Secret with filtered key", func(t *testing.T) {
			name := RandomizedSuffixName("config-secret-key-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/config-secret-key-route.yaml",
				"--name", name,
				"--config", "secret:my-sec-multi/my-secret-key-2",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).ShouldNot(ContainSubstring(secDataMulti["my-secret-key"]))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(secDataMulti["my-secret-key-2"]))
		})

	})
}

func TestRunConfigBuildProperties(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// will be used by a second test
		var nameBuildTimeProperty string
		t.Run("Build time property", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-route")
			nameBuildTimeProperty = name
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-route.yaml",
				"--name", name,
				"--build-property", "quarkus.application.name=my-super-application",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-super-application"))
			// Don't delete - we need it for next test execution
		})

		// We need to check also that the property (which is available in the IntegrationKit) is correctly replaced and that we don't reuse the same kit
		t.Run("Build time property updated", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-route-updated")
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-route.yaml",
				"--name", name,
				"--build-property", "quarkus.application.name=my-super-application-updated",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-super-application-updated"))
			// Verify the integration kits are different
			g.Eventually(IntegrationKitName(t, ctx, ns, name)).ShouldNot(Equal(IntegrationKitName(t, ctx, ns, nameBuildTimeProperty)()))
		})

		// Build-Properties file
		t.Run("Build time property file", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-file-route")
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml",
				"--name", name,
				"--build-property", "file:./files/quarkus.properties",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-super-application"))
		})

		t.Run("Build time property file with precedence", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-file-route-precedence")
			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml",
				"--name", name,
				"--build-property", "quarkus.application.name=my-overridden-application",
				"--build-property", "file:./files/quarkus.properties",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-overridden-application"))
		})

		t.Run("Build time property from ConfigMap", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-file-route-cm")
			var cmData = make(map[string]string)
			cmData["quarkus.application.name"] = "my-cool-application"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-build-property", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml",
				"--name", name,
				"--build-property", "configmap:my-cm-test-build-property",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-cool-application"))
		})

		t.Run("Build time property from ConfigMap as property file", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-file-route-cm-prop")
			var cmData = make(map[string]string)
			cmData["my.properties"] = "quarkus.application.name=my-super-cool-application"
			err := CreatePlainTextConfigmap(t, ctx, ns, "my-cm-test-build-properties", cmData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml",
				"--name", name,
				"--build-property", "configmap:my-cm-test-build-properties",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-super-cool-application"))
		})

		t.Run("Build time property from Secret", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-file-route-sec")
			var secData = make(map[string]string)
			secData["quarkus.application.name"] = "my-great-application"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-build-property", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml",
				"--name", name,
				"--build-property", "secret:my-sec-test-build-property",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-great-application"))
		})

		t.Run("Build time property from Secret as property file", func(t *testing.T) {
			name := RandomizedSuffixName("build-property-file-route-sec-prop")
			var secData = make(map[string]string)
			secData["my.properties"] = "quarkus.application.name=my-awesome-application"
			err := CreatePlainTextSecret(t, ctx, ns, "my-sec-test-build-properties", secData)
			g.Expect(err).To(BeNil())

			g.Expect(KamelRun(t, ctx, ns, "./files/build-property-file-route.yaml",
				"--name", name,
				"--build-property", "secret:my-sec-test-build-properties").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("my-awesome-application"))
		})

	})
}
