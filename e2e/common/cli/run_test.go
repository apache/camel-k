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
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

var sampleJar = "https://raw.githubusercontent.com/apache/camel-k/main/e2e/common/traits/files/jvm/sample-1.0.jar"

func TestKamelCLIRun(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Run and update", func(t *testing.T) {
			name := RandomizedSuffixName("run")
			g.Expect(KamelRun(t, ctx, ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magic default"))

			// Re-run the Integration with an updated configuration
			g.Expect(KamelRun(t, ctx, ns, "files/run.yaml", "--name", name, "-p", "property=value").Execute()).
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
		})

		t.Run("Run with glob patterns", func(t *testing.T) {
			t.Run("YAML", func(t *testing.T) {
				name := RandomizedSuffixName("run")
				g.Expect(KamelRun(t, ctx, ns, "files/glob/run*", "--name", name).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello run 1 default"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello run 2 default"))
			})

			t.Run("Java", func(t *testing.T) {
				name := RandomizedSuffixName("java")
				g.Expect(KamelRun(t, ctx, ns, "files/glob/Java*", "--name", name).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello java 1 default"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello java 2 default"))
			})

			t.Run("All", func(t *testing.T) {
				name := RandomizedSuffixName("java")
				g.Expect(KamelRun(t, ctx, ns, "files/glob/*", "--name", name).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello run 1 default"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello run 2 default"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello java 1 default"))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello java 2 default"))
				g.Eventually(DeleteIntegrations(t, ctx, ns), TestTimeoutLong).Should(Equal(0))
			})

		})
	})
}
