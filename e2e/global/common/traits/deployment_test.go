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
	appsv1 "k8s.io/api/apps/v1"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestRecreateDeploymentStrategyTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-jolokia"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("Run with Recreate Deployment Strategy", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"-t", "deployment.strategy="+string(appsv1.RecreateDeploymentStrategyType)).
				Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Eventually(Deployment(ns, "java"), TestTimeoutMedium).Should(PointTo(MatchFields(IgnoreExtras,
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

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}

func TestRollingUpdateDeploymentStrategyTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-jolokia"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("Run with RollingUpdate Deployment Strategy", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"-t", "deployment.strategy="+string(appsv1.RollingUpdateDeploymentStrategyType)).
				Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Eventually(Deployment(ns, "java"), TestTimeoutMedium).Should(PointTo(MatchFields(IgnoreExtras,
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

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
