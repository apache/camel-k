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

package builder

import (
	"os"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestRunGlobalInstall(t *testing.T) {
	forceGlobalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
	if !forceGlobalTest {
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)
		if ocp {
			t.Skip("Prefer not to run on OpenShift to avoid giving more permissions to the user running tests")
			return
		}
	}

	WithGlobalOperatorNamespace(t, func(operatorNamespace string) {
		Expect(Kamel("install", "-n", operatorNamespace, "--global", "--force").Execute()).To(Succeed())
		Eventually(OperatorPodPhase(operatorNamespace), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		t.Run("Global test on namespace with platform", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns2 string) {
				// Creating platform
				Expect(Kamel("install", "-n", ns2, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())

				Expect(Kamel("run", "-n", ns2, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns2, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns2, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(ns2, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(ns2 + "\\/.*"))
				kit := IntegrationKit(ns2, "java")()
				Expect(Kamel("delete", "--all", "-n", ns2).Execute()).To(Succeed())
				Expect(Kits(ns2)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit)))
				Expect(Kits(operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))

				Expect(Lease(ns2, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		t.Run("Global test on namespace with its own operator", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns3 string) {
				if NoOlmOperatorImage != "" {
					Expect(Kamel("install", "-n", ns3, "--olm=false", "--operator-image", NoOlmOperatorImage).Execute()).To(Succeed())
				} else {
					Expect(Kamel("install", "-n", ns3, "--olm=false").Execute()).To(Succeed())
				}
				Eventually(OperatorPodPhase(ns3), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Expect(Kamel("run", "-n", ns3, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns3, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns3, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(ns3, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(ns3 + "\\/.*"))
				kit := IntegrationKit(ns3, "java")()
				Expect(Kits(ns3)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit)))
				Expect(Kamel("delete", "--all", "-n", ns3).Execute()).To(Succeed())

				Expect(Lease(ns3, platform.OperatorLockName)()).ShouldNot(BeNil(),
					"Controller Runtime is expected to use Leases for leader election: if this changes we should update our locking logic",
				)
			})
		})

		t.Run("Global test on namespace without platform", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns4 string) {
				Expect(Kamel("run", "-n", ns4, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns4, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns4, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(ns4, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(operatorNamespace + "\\/.*"))
				kit := IntegrationKit(ns4, "java")()
				Expect(Kamel("delete", "--all", "-n", ns4).Execute()).To(Succeed())
				Expect(Kits(ns4)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))
				Expect(Kits(operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // Kit built globally

				Expect(Lease(ns4, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		t.Run("Global test on namespace without platform with external kit", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns5 string) {
				Expect(Kamel("run", "-n", ns5, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns5, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns5, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(ns5, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(operatorNamespace + "\\/.*"))
				kit := IntegrationKit(ns5, "java")()
				Expect(Kamel("delete", "--all", "-n", ns5).Execute()).To(Succeed())
				Expect(Kits(ns5)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))
				globalKits := Kits(operatorNamespace)()
				Expect(globalKits).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // Reusing the same global kit

				// external kit mirroring the global one
				externalKit := v1.IntegrationKit{
					ObjectMeta: metav1.ObjectMeta{
						Namespace: ns5,
						Name:      "external",
						Labels: map[string]string{
							"camel.apache.org/kit.type": v1.IntegrationKitTypeExternal,
						},
					},
					Spec: v1.IntegrationKitSpec{
						Image: getKitImage(operatorNamespace, kit),
					},
				}
				Expect(TestClient().Create(TestContext, &externalKit)).Should(BeNil())

				Expect(Kamel("run", "-n", ns5, "files/Java.java", "--name", "ext", "--kit", "external").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns5, "ext"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns5, "ext"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationKit(ns5, "ext")()).Should(Equal("external"))
				Expect(Kamel("delete", "--all", "-n", ns5).Execute()).To(Succeed())
				Expect(Kits(ns5)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement("external")))        // the external one
				Expect(Kits(operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // the global one

				Expect(Lease(ns5, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		Expect(Kamel("uninstall", "-n", operatorNamespace, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())
	})
}

func integrationKitsToNamesTransform() func([]v1.IntegrationKit) []string {
	return func(iks []v1.IntegrationKit) []string {
		var names []string
		for _, x := range iks {
			names = append(names, x.Name)
		}
		return names
	}
}

func getKitImage(ns string, name string) string {
	get := v1.IntegrationKit{
		TypeMeta: metav1.TypeMeta{
			Kind:       "IntegrationKit",
			APIVersion: v1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	key := ctrl.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	if err := TestClient().Get(TestContext, key, &get); err != nil {
		return ""
	}
	return get.Status.Image
}
