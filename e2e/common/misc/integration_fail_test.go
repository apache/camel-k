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
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBadRouteIntegration(t *testing.T) {
	RegisterTestingT(t)

	t.Run("run bad java route", func(t *testing.T) {
		name := RandomizedSuffixName("bad-route")
		Expect(KamelRunWithID(operatorID, ns, "files/BadRoute.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionFalse))

		// Make sure the Integration can be scaled
		Expect(ScaleIntegration(ns, name, 2)).To(Succeed())
		// Check the scale cascades into the Deployment scale
		Eventually(IntegrationPods(ns, name), TestTimeoutShort).Should(HaveLen(2))
		// Check it also cascades into the Integration scale subresource Status field
		Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 2)))
		// Check the Integration stays in error phase
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))

		// Kit valid
		kitName := IntegrationKit(ns, name)()
		integrationKitNamespace := IntegrationKitNamespace(ns, name)()
		Eventually(KitPhase(integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
	})

	t.Run("run missing dependency java route", func(t *testing.T) {
		name := RandomizedSuffixName("java-route")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
			"-d", "mvn:com.example:nonexistent:1.0").Execute()).To(Succeed())
		// Integration in error
		Eventually(IntegrationPhase(ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).Should(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionKitAvailableReason)))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionKitAvailable), TestTimeoutShort).Should(
			WithTransform(IntegrationConditionMessage, ContainSubstring("is in state \"Error\"")))
		// Kit in error
		kitName := IntegrationKit(ns, name)()
		integrationKitNamespace := IntegrationKitNamespace(ns, name)()
		Eventually(KitPhase(integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseError))
		//Build in error with 5 attempts
		build := Build(integrationKitNamespace, kitName)()
		Eventually(build.Status.Phase, TestTimeoutShort).Should(Equal(v1.BuildPhaseError))
		Eventually(build.Status.Failure.Recovery.Attempt, TestTimeoutShort).Should(Equal(5))

		// Fixing the route should reconcile the Integration
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPhase(ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseRunning))
		// New Kit success
		kitRecoveryName := IntegrationKit(ns, name)()
		integrationKitRecoveryNamespace := IntegrationKitNamespace(ns, name)()
		Eventually(KitPhase(integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
		Expect(kitRecoveryName).NotTo(Equal(kitName))
		// New Build success
		buildRecovery := Build(integrationKitRecoveryNamespace, kitRecoveryName)()
		Eventually(buildRecovery.Status.Phase, TestTimeoutShort).Should(Equal(v1.BuildPhaseSucceeded))

	})

	t.Run("run invalid dependency java route", func(t *testing.T) {
		name := RandomizedSuffixName("invalid-dependency")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
			"-d", "camel:non-existent").Execute()).To(Succeed())
		// Integration in error with Initialization Failed condition
		Eventually(IntegrationPhase(ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(And(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionInitializationFailedReason)),
			WithTransform(IntegrationConditionMessage, HavePrefix("error during trait customization")),
		))
		// Kit shouldn't be created
		Consistently(IntegrationKit(ns, name), 10*time.Second).Should(BeEmpty())

		// Fixing the route should reconcile the Integration in Initialization Failed condition to Running
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		// New Kit success
		kitRecoveryName := IntegrationKit(ns, name)()
		integrationKitRecoveryNamespace := IntegrationKitNamespace(ns, name)()
		Eventually(KitPhase(integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
		// New Build success
		buildRecovery := Build(integrationKitRecoveryNamespace, kitRecoveryName)()
		Eventually(buildRecovery.Status.Phase, TestTimeoutShort).Should(Equal(v1.BuildPhaseSucceeded))
	})

	t.Run("run unresolvable component java route", func(t *testing.T) {
		name := RandomizedSuffixName("unresolvable-route")
		Expect(KamelRunWithID(operatorID, ns, "files/Unresolvable.java", "--name", name).Execute()).To(Succeed())
		// Integration in error with Initialization Failed condition
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(And(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionInitializationFailedReason)),
			WithTransform(IntegrationConditionMessage, HavePrefix("error during trait customization")),
		))
		// Kit shouldn't be created
		Consistently(IntegrationKit(ns, name), 10*time.Second).Should(BeEmpty())

		// Fixing the route should reconcile the Integration in Initialization Failed condition to Running
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		// New Kit success
		kitRecoveryName := IntegrationKit(ns, name)()
		integrationKitRecoveryNamespace := IntegrationKitNamespace(ns, name)()
		Eventually(KitPhase(integrationKitRecoveryNamespace, kitRecoveryName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))
		// New Build success
		buildRecovery := Build(integrationKitRecoveryNamespace, kitRecoveryName)()
		Eventually(buildRecovery.Status.Phase, TestTimeoutShort).Should(Equal(v1.BuildPhaseSucceeded))
	})

	t.Run("run invalid java route", func(t *testing.T) {
		name := RandomizedSuffixName("invalid-java-route")
		Expect(KamelRunWithID(operatorID, ns, "files/InvalidJava.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Compilation error"))

		// Kit valid
		kitName := IntegrationKit(ns, name)()
		integrationKitNamespace := IntegrationKitNamespace(ns, name)()
		Eventually(KitPhase(integrationKitNamespace, kitName), TestTimeoutShort).Should(Equal(v1.IntegrationKitPhaseReady))

		// Fixing the route should reconcile the Integration in Initialization Failed condition to Running
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Kit should not have changed
		kitRecoveryName := IntegrationKit(ns, name)()
		Expect(kitRecoveryName).To(Equal(kitName))

	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
