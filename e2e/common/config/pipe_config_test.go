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

// Tests on pipe with kamelets containing configuration from properties and secrets.
func TestPipeConfig(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("test custom source/sink pipe", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "my-pipe-timer-source")()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, "my-pipe-log-sink")()).To(Succeed())
			t.Run("run test default config using properties", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-properties")

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-p", "source.message=My pipe message",
					"-p", "sink.loggerName=myPipeLogger",
					"--name", name,
				).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeLogger"))
			})

			t.Run("run test implicit default config using labeled secret", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-default-implicit-secret")
				secretName := "my-pipe-default-implicit-secret"

				var secData = make(map[string]string)
				secData["camel.kamelet.my-pipe-timer-source.message"] = "My pipe secret message"
				var labels = make(map[string]string)
				labels["camel.apache.org/kamelet"] = "my-pipe-timer-source"
				g.Expect(CreatePlainTextSecretWithLabels(t, ctx, ns, secretName, secData, labels)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-p", "sink.loggerName=myDefaultLogger",
					"--name", name,
				).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe secret message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myDefaultLogger"))

				g.Expect(DeleteSecret(t, ctx, ns, secretName)).To(Succeed())
			})

			t.Run("run test implicit default config using mounted secret", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-default-implicit-secret")
				secretName := "my-pipe-default-implicit-secret"

				var secData = make(map[string]string)
				secData["camel.kamelet.my-pipe-timer-source.message"] = "My pipe secret message"
				secData["camel.kamelet.my-pipe-log-sink.loggerName"] = "myPipeSecretLogger"
				g.Expect(CreatePlainTextSecret(t, ctx, ns, secretName, secData)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-t", "mount.configs=secret:"+secretName,
					"--name", name,
				).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe secret message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeSecretLogger"))

				g.Expect(DeleteSecret(t, ctx, ns, secretName)).To(Succeed())
			})

			t.Run("run test implicit default config using mounted configmap", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-default-implicit-configmap")
				cmName := "my-pipe-default-implicit-configmap"

				var cmData = make(map[string]string)
				cmData["camel.kamelet.my-pipe-timer-source.message"] = "My pipe configmap message"
				cmData["camel.kamelet.my-pipe-log-sink.loggerName"] = "myPipeConfigmapLogger"
				g.Expect(CreatePlainTextConfigmap(t, ctx, ns, cmName, cmData)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-t", "mount.configs=configmap:"+cmName,
					"--name", name,
				).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe configmap message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeConfigmapLogger"))

				g.Expect(DeleteConfigmap(t, ctx, ns, cmName)).To(Succeed())
			})

			t.Run("run test implicit named config using mounted secret", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-named-implicit-secret")
				secretName := "my-pipe-named-implicit-secret"

				var secData = make(map[string]string)
				secData["camel.kamelet.my-pipe-timer-source.mynamedconfig.message"] = "My pipe named secret message"
				secData["camel.kamelet.my-pipe-log-sink.mynamedconfig.loggerName"] = "myPipeNamedSecretLogger"
				g.Expect(CreatePlainTextSecret(t, ctx, ns, secretName, secData)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-t", "mount.configs=secret:"+secretName,
					"-p", "source.message={{mynamedconfig.message}}",
					"-p", "sink.loggerName={{mynamedconfig.loggerName}}",
					"--name", name,
				).Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe named secret message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeNamedSecretLogger"))

				g.Expect(DeleteSecret(t, ctx, ns, secretName)).To(Succeed())
			})

			t.Run("run test implicit named config using mounted configmap", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-named-implicit-configmap")
				cmName := "my-pipe-named-implicit-configmap"

				var cmData = make(map[string]string)
				cmData["camel.kamelet.my-pipe-timer-source.mynamedconfig.message"] = "My pipe named configmap message"
				cmData["camel.kamelet.my-pipe-log-sink.mynamedconfig.loggerName"] = "myPipeNamedConfigmapLogger"
				g.Expect(CreatePlainTextConfigmap(t, ctx, ns, cmName, cmData)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-t", "mount.configs=configmap:"+cmName,
					"-p", "source.message={{mynamedconfig.message}}",
					"-p", "sink.loggerName={{mynamedconfig.loggerName}}",
					"--name", name,
				).Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe named configmap message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeNamedConfigmapLogger"))

				g.Expect(DeleteConfigmap(t, ctx, ns, cmName)).To(Succeed())
			})
			t.Run("run test implicit specific config using mounted secret", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-specific-secret")
				secretName := "my-pipe-specific-secret"

				var secData = make(map[string]string)
				secData["mynamedconfig.message"] = "My pipe specific secret message"
				secData["mynamedconfig.loggerName"] = "myPipeSpecificSecretLogger"
				g.Expect(CreatePlainTextSecret(t, ctx, ns, secretName, secData)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-t", "mount.configs=secret:"+secretName,
					"-p", "source.message={{mynamedconfig.message}}",
					"-p", "sink.loggerName={{mynamedconfig.loggerName}}",
					"--name", name,
				).Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe specific secret message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeSpecificSecretLogger"))

				g.Expect(DeleteSecret(t, ctx, ns, secretName)).To(Succeed())
			})
			t.Run("run test implicit specific config using mounted configmap", func(t *testing.T) {
				name := RandomizedSuffixName("my-pipe-with-specific-configmap")
				cmName := "my-pipe-specific-configmap"

				var cmData = make(map[string]string)
				cmData["mynamedconfig.message"] = "My pipe specific configmap message"
				cmData["mynamedconfig.loggerName"] = "myPipeSpecificConfgmapLogger"
				g.Expect(CreatePlainTextConfigmap(t, ctx, ns, cmName, cmData)).To(Succeed())

				g.Expect(KamelBind(t, ctx, ns,
					"my-pipe-timer-source",
					"my-pipe-log-sink",
					"-t", "mount.configs=configmap:"+cmName,
					"-p", "source.message={{mynamedconfig.message}}",
					"-p", "sink.loggerName={{mynamedconfig.loggerName}}",
					"--name", name,
				).Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("My pipe specific configmap message"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("myPipeSpecificConfgmapLogger"))

				g.Expect(DeleteConfigmap(t, ctx, ns, cmName)).To(Succeed())
			})
		})
	})
}
