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
	"fmt"
	"strings"
	"testing"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestRunGlobalInstall(t *testing.T) {
	RegisterTestingT(t)

	WithGlobalOperatorNamespace(t, func(operatorNamespace string) {
		Expect(KamelInstall(t, operatorNamespace, "--global", "--force").Execute()).To(Succeed())
		Eventually(OperatorPodPhase(t, operatorNamespace), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		t.Run("Global CamelCatalog reconciliation", func(t *testing.T) {
			Eventually(Platform(t, operatorNamespace)).ShouldNot(BeNil())
			Eventually(PlatformConditionStatus(t, operatorNamespace, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
			Eventually(CamelCatalog(t, operatorNamespace, catalogName)).ShouldNot(BeNil())
			Eventually(CamelCatalogPhase(t, operatorNamespace, catalogName), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
		})

		t.Run("Global test on namespace with platform", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns2 string) {
				// Creating namespace local platform
				Expect(KamelInstall(t, ns2, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())
				Eventually(Platform(t, ns2)).ShouldNot(BeNil())

				// Run with global operator id
				Expect(KamelRun(t, ns2, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(t, ns2, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(t, ns2, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(t, ns2, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(ns2 + "\\/.*"))
				kit := IntegrationKit(t, ns2, "java")()
				Expect(Kamel(t, "delete", "--all", "-n", ns2).Execute()).To(Succeed())
				Expect(Kits(t, ns2)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit)))
				Expect(Kits(t, operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))

				Expect(Lease(t, ns2, platform.DefaultPlatformName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		t.Run("Global test on namespace with its own operator", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns3 string) {
				operatorID := "camel-k-local-ns3"
				if NoOlmOperatorImage != "" {
					Expect(KamelInstallWithID(t, operatorID, ns3, "--olm=false", "--operator-image", NoOlmOperatorImage).Execute()).To(Succeed())
				} else {
					Expect(KamelInstallWithID(t, operatorID, ns3, "--olm=false").Execute()).To(Succeed())
				}
				Eventually(OperatorPodPhase(t, ns3), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Expect(KamelRunWithID(t, operatorID, ns3, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(t, ns3, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(t, ns3, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(t, ns3, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(ns3 + "\\/.*"))
				kit := IntegrationKit(t, ns3, "java")()
				Expect(Kits(t, ns3)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit)))
				Expect(Kamel(t, "delete", "--all", "-n", ns3).Execute()).To(Succeed())

				Expect(Lease(t, ns3, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
				Expect(Lease(t, ns3, platform.GetOperatorLockName(operatorID))()).ShouldNot(BeNil(),
					"Controller Runtime is expected to use Leases for leader election: if this changes we should update our locking logic",
				)
			})
		})

		t.Run("Global test on namespace without platform", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns4 string) {
				Expect(KamelRun(t, ns4, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(t, ns4, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(t, ns4, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(t, ns4, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(operatorNamespace + "\\/.*"))
				kit := IntegrationKit(t, ns4, "java")()
				Expect(Kamel(t, "delete", "--all", "-n", ns4).Execute()).To(Succeed())
				Expect(Kits(t, ns4)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))
				Expect(Kits(t, operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // Kit built globally

				Expect(Lease(t, ns4, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		t.Run("Global test on namespace without platform with external kit", func(t *testing.T) {
			WithNewTestNamespace(t, func(ns5 string) {
				Expect(KamelRun(t, ns5, "files/Java.java").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(t, ns5, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(t, ns5, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationConditionMessage(IntegrationCondition(t, ns5, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(operatorNamespace + "\\/.*"))
				kit := IntegrationKit(t, ns5, "java")()
				Expect(Kamel(t, "delete", "--all", "-n", ns5).Execute()).To(Succeed())
				Expect(Kits(t, ns5)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))
				globalKits := Kits(t, operatorNamespace)()
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
						Image: getKitImage(t, operatorNamespace, kit),
					},
				}
				Expect(TestClient(t).Create(TestContext, &externalKit)).Should(BeNil())

				Expect(KamelRun(t, ns5, "files/Java.java", "--name", "ext", "--kit", "external", "-t", "jvm.enabled=true").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(t, ns5, "ext"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(t, ns5, "ext"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				Expect(IntegrationKit(t, ns5, "ext")()).Should(Equal("external"))
				Expect(Kamel(t, "delete", "--all", "-n", ns5).Execute()).To(Succeed())
				Expect(Kits(t, ns5)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement("external")))        // the external one
				Expect(Kits(t, operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // the global one

				Expect(Lease(t, ns5, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		Expect(Kamel(t, "uninstall", "-n", operatorNamespace, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())
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

func getKitImage(t *testing.T, ns string, name string) string {
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
	if err := TestClient(t).Get(TestContext, key, &get); err != nil {
		return ""
	}
	return get.Status.Image
}
