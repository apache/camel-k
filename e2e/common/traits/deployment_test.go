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

package traits

import (
	"context"
	"os/exec"
	"testing"

	appsv1 "k8s.io/api/apps/v1"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestRecreateDeploymentStrategyTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Run with Recreate Deployment Strategy", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "deployment.strategy="+string(appsv1.RecreateDeploymentStrategyType)).
				Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Eventually(Deployment(t, ctx, ns, name), TestTimeoutMedium).Should(PointTo(MatchFields(IgnoreExtras,
				Fields{
					"Spec": MatchFields(IgnoreExtras,
						Fields{
							"Strategy": MatchFields(IgnoreExtras,
								Fields{
									"Type": Equal(appsv1.RecreateDeploymentStrategyType),
								}),
						}),
				}),
			))
		})
	})
}

func TestRollingUpdateDeploymentStrategyTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Run with RollingUpdate Deployment Strategy", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "-t", "deployment.strategy="+string(appsv1.RollingUpdateDeploymentStrategyType)).
				Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Eventually(Deployment(t, ctx, ns, "java"), TestTimeoutMedium).Should(PointTo(MatchFields(IgnoreExtras,
				Fields{
					"Spec": MatchFields(IgnoreExtras,
						Fields{
							"Strategy": MatchFields(IgnoreExtras,
								Fields{
									"Type": Equal(appsv1.RollingUpdateDeploymentStrategyType),
								}),
						}),
				}),
			))
		})
	})
}

func TestDeploymentFailureShouldReportIntegrationCondition(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		nsRestr := "restr"
		// Create restricted namespace
		ExpectExecSucceed(t, g,
			exec.Command(
				"kubectl",
				"create",
				"ns",
				nsRestr,
			),
		)
		ExpectExecSucceed(t, g,
			exec.Command(
				"kubectl",
				"label",
				"--overwrite",
				"ns",
				nsRestr,
				"pod-security.kubernetes.io/enforce=baseline",
				"pod-security.kubernetes.io/enforce-version=latest",
				"pod-security.kubernetes.io/enforce=restricted",
				"pod-security.kubernetes.io/warn-version=latest",
				"pod-security.kubernetes.io/audit=restricted",
				"pod-security.kubernetes.io/audit-version=latest",
			),
		)
		// Create an Integration into a restricted namespace
		name := RandomizedSuffixName("java-fail")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-n", nsRestr).Execute()).To(Succeed())
		// Check the error is reported into the Integration
		g.Eventually(IntegrationPhase(t, ctx, nsRestr, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseError))
		g.Eventually(IntegrationCondition(t, ctx, nsRestr, name, v1.IntegrationConditionReady)().Status).
			Should(Equal(corev1.ConditionFalse))
		g.Eventually(IntegrationCondition(t, ctx, nsRestr, name, v1.IntegrationConditionReady)().Message).
			Should(ContainSubstring("is forbidden: violates PodSecurity"))
	})
}
