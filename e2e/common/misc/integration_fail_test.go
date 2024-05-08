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

package misc

import (
	"context"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBadRouteIntegration(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-bad-route"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("run bad java route", func(t *testing.T) {
			name := RandomizedSuffixName("bad-route")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/BadRoute.java", "--name", name, "-t", "health.enabled=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))

			// Make sure the Integration can be scaled
			g.Expect(ScaleIntegration(t, ctx, ns, name, 2)).To(Succeed())
			// Check the scale cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(2))
			// Check it also cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Check the Integration stays in error phase
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))

			// Kit valid
			kitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
		})

		t.Run("run missing dependency java route", func(t *testing.T) {
			name := RandomizedSuffixName("java-route")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name,
				"-d", "mvn:com.example:nonexistent:1.0", "-t", "health.enabled=false").Execute()).To(Succeed())
			// Integration in error
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).Should(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionKitAvailableReason)))
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).Should(
				WithTransform(IntegrationConditionMessage, ContainSubstring("is in state \"Error\"")))
			// Kit in error
			kitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseError))
			//Build in error with 5 attempts
			g.Eventually(BuildPhase(t, ctx, integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.BuildPhaseError))
			g.Eventually(BuildFailureRecoveryAttempt(t, ctx, integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(5))

			// Fixing the route should reconcile the Integration
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseRunning))
			// New Kit success
			kitRecoveryName := IntegrationKit(t, ctx, ns, name)()
			integrationKitRecoveryNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
			g.Expect(kitRecoveryName).NotTo(Equal(kitName))
			// New Build success
			g.Eventually(BuildPhase(t, ctx, integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.BuildPhaseSucceeded))

		})

		t.Run("run invalid dependency java route", func(t *testing.T) {
			name := RandomizedSuffixName("invalid-dependency")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-d", "camel:non-existent", "-t", "health.enabled=false").Execute()).To(Succeed())
			// Integration in error with Initialization Failed condition
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionInitializationFailedReason)),
				WithTransform(IntegrationConditionMessage, HavePrefix("error during trait customization")),
			))
			// Kit shouldn't be created
			g.Consistently(IntegrationKit(t, ctx, ns, name), 10*time.Second).Should(BeEmpty())

			// Fixing the route should reconcile the Integration in Initialization Failed condition to Running
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// New Kit success
			kitRecoveryName := IntegrationKit(t, ctx, ns, name)()
			integrationKitRecoveryNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
			// New Build success
			g.Eventually(BuildPhase(t, ctx, integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.BuildPhaseSucceeded))
		})

		t.Run("run unresolvable component java route", func(t *testing.T) {
			name := RandomizedSuffixName("unresolvable-route")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Unresolvable.java", "--name", name, "-t", "health.enabled=false").Execute()).To(Succeed())
			// Integration in error with Initialization Failed condition
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionInitializationFailedReason)),
				WithTransform(IntegrationConditionMessage, HavePrefix("error during trait customization")),
			))
			// Kit shouldn't be created
			g.Consistently(IntegrationKit(t, ctx, ns, name), 10*time.Second).Should(BeEmpty())

			// Fixing the route should reconcile the Integration in Initialization Failed condition to Running
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "health.enabled=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// New Kit success
			kitRecoveryName := IntegrationKit(t, ctx, ns, name)()
			integrationKitRecoveryNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
			// New Build success
			g.Eventually(BuildPhase(t, ctx, integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.BuildPhaseSucceeded))
		})

		t.Run("run invalid java route", func(t *testing.T) {
			name := RandomizedSuffixName("invalid-java-route")
			// Skip the health check so we can quickly read from log
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/InvalidJava.java", "--name", name, "-t", "health.enabled=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Compilation error"))

			// Kit valid
			kitName := IntegrationKit(t, ctx, ns, name)()
			integrationKitNamespace := IntegrationKitNamespace(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))

			// Fixing the route should reconcile the Integration in Initialization Failed condition to Running
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "health.enabled=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Kit should not have changed
			kitRecoveryName := IntegrationKit(t, ctx, ns, name)()
			g.Expect(kitRecoveryName).To(Equal(kitName))

		})

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
