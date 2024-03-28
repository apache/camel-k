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

package cli

import (
	"context"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// Smoke tests on usage of jbang camel k plugin
func TestCamelKCLIRun(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {

		t.Run("Run and update", func(t *testing.T) {
			name := RandomizedSuffixName("run")
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magic default"))

			// Re-run the Integration with an updated configuration
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/run.yaml", "--name", name, "-p", "property=value").Execute()).
				To(Succeed())

			// Check the Deployment has progressed successfully
			g.Eventually(DeploymentCondition(t, ctx, ns, name, appsv1.DeploymentProgressing), TestTimeoutShort).
				Should(MatchFields(IgnoreExtras, Fields{
					"Status": Equal(corev1.ConditionTrue),
					"Reason": Equal("NewReplicaSetAvailable"),
				}))

			// Check the new configuration is taken into account
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magic value"))

			// Clean up
			g.Eventually(DeleteIntegrations(t, ctx, ns), TestTimeoutLong).Should(Equal(0))
		})

	})

}

func TestCamelKCLIDelete(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("delete running integration", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(CamelK(t, ctx, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete building integration", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(CamelK(t, ctx, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete several integrations", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(CamelK(t, ctx, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
			g.Expect(CamelK(t, ctx, "delete", "java", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "java")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "java"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete all integrations", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
			g.Eventually(Integration(t, ctx, ns, "java")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "java"), TestTimeoutLong).Should(BeNil())
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

}

func TestCamelKCLILog(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("check integration log", func(t *testing.T) {
			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml", "--name", "log-yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "log-yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			// first line of the integration logs
			firstLine := strings.Split(IntegrationLogs(t, ctx, ns, "log-yaml")(), "\n")[0]

			logsCLI := GetOutputStringAsync(CamelK(t, ctx, "logs", "log-yaml", "-n", ns))
			g.Eventually(logsCLI).Should(ContainSubstring(firstLine))

			logs := strings.Split(IntegrationLogs(t, ctx, ns, "log-yaml")(), "\n")
			lastLine := logs[len(logs)-1]

			logsCLI = GetOutputStringAsync(CamelK(t, ctx, "log", "log-yaml", "-n", ns, "--tail", "5"))
			g.Eventually(logsCLI).Should(ContainSubstring(lastLine))
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

}

// TODO Remove before merge
func TestConf(t *testing.T) {
	val, present := os.LookupEnv("JBANG_NO_VERSION_CHECK")
	assert.True(t, present)
	assert.Equal(t, "true", val)
}
