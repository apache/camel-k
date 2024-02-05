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
	WithGlobalOperatorNamespace(t, func(ctx context.Context, g *WithT, operatorNamespace string) {
		g.Expect(KamelInstall(t, ctx, operatorNamespace, "--global", "--force")).To(Succeed())
		g.Eventually(OperatorPodPhase(t, ctx, operatorNamespace), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		t.Run("Global CamelCatalog reconciliation", func(t *testing.T) {
			g.Eventually(Platform(t, ctx, operatorNamespace)).ShouldNot(BeNil())
			g.Eventually(PlatformConditionStatus(t, ctx, operatorNamespace, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
			g.Eventually(CamelCatalog(t, ctx, operatorNamespace, catalogName)).ShouldNot(BeNil())
			g.Eventually(CamelCatalogPhase(t, ctx, operatorNamespace, catalogName), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
		})

		t.Run("Global test on namespace with platform", func(t *testing.T) {
			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns2 string) {
				// Creating namespace local platform
				g.Expect(KamelInstall(t, ctx, ns2, "--skip-operator-setup", "--olm=false")).To(Succeed())
				g.Eventually(Platform(t, ctx, ns2)).ShouldNot(BeNil())

				// Run with global operator id
				g.Expect(KamelRun(t, ctx, ns2, "files/Java.java").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns2, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns2, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				g.Expect(IntegrationConditionMessage(IntegrationCondition(t, ctx, ns2, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(ns2 + "\\/.*"))
				kit := IntegrationKit(t, ctx, ns2, "java")()
				g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns2).Execute()).To(Succeed())
				g.Expect(Kits(t, ctx, ns2)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit)))
				g.Expect(Kits(t, ctx, operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))

				g.Expect(Lease(t, ctx, ns2, platform.DefaultPlatformName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		t.Run("Global test on namespace with its own operator", func(t *testing.T) {
			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns3 string) {
				operatorID := "camel-k-local-ns3"
				if NoOlmOperatorImage != "" {
					g.Expect(KamelInstallWithID(t, ctx, operatorID, ns3, "--olm=false", "--operator-image", NoOlmOperatorImage)).To(Succeed())
				} else {
					g.Expect(KamelInstallWithID(t, ctx, operatorID, ns3, "--olm=false")).To(Succeed())
				}
				g.Eventually(OperatorPodPhase(t, ctx, ns3), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Expect(KamelRunWithID(t, ctx, operatorID, ns3, "files/Java.java").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns3, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns3, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				g.Expect(IntegrationConditionMessage(IntegrationCondition(t, ctx, ns3, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(ns3 + "\\/.*"))
				kit := IntegrationKit(t, ctx, ns3, "java")()
				g.Expect(Kits(t, ctx, ns3)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit)))
				g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns3).Execute()).To(Succeed())

				g.Expect(Lease(t, ctx, ns3, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
				g.Expect(Lease(t, ctx, ns3, platform.GetOperatorLockName(operatorID))()).ShouldNot(BeNil(),
					"Controller Runtime is expected to use Leases for leader election: if this changes we should update our locking logic",
				)
			})
		})

		t.Run("Global test on namespace without platform", func(t *testing.T) {
			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns4 string) {
				g.Expect(KamelRun(t, ctx, ns4, "files/Java.java").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns4, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns4, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				g.Expect(IntegrationConditionMessage(IntegrationCondition(t, ctx, ns4, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(operatorNamespace + "\\/.*"))
				kit := IntegrationKit(t, ctx, ns4, "java")()
				g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns4).Execute()).To(Succeed())
				g.Expect(Kits(t, ctx, ns4)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))
				g.Expect(Kits(t, ctx, operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // Kit built globally

				g.Expect(Lease(t, ctx, ns4, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		t.Run("Global test on namespace without platform with external kit", func(t *testing.T) {
			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns5 string) {
				g.Expect(KamelRun(t, ctx, ns5, "files/Java.java").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns5, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns5, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				g.Expect(IntegrationConditionMessage(IntegrationCondition(t, ctx, ns5, "java", v1.IntegrationConditionPlatformAvailable)())).To(MatchRegexp(operatorNamespace + "\\/.*"))
				kit := IntegrationKit(t, ctx, ns5, "java")()
				g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns5).Execute()).To(Succeed())
				g.Expect(Kits(t, ctx, ns5)()).Should(WithTransform(integrationKitsToNamesTransform(), Not(ContainElement(kit))))
				globalKits := Kits(t, ctx, operatorNamespace)()
				g.Expect(globalKits).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // Reusing the same global kit

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
						Image: getKitImage(t, ctx, operatorNamespace, kit),
					},
				}
				g.Expect(TestClient(t).Create(ctx, &externalKit)).Should(BeNil())

				g.Expect(KamelRun(t, ctx, ns5, "files/Java.java", "--name", "ext", "--kit", "external", "-t", "jvm.enabled=true").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns5, "ext"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns5, "ext"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
				g.Expect(IntegrationKit(t, ctx, ns5, "ext")()).Should(Equal("external"))
				g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns5).Execute()).To(Succeed())
				g.Expect(Kits(t, ctx, ns5)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement("external")))        // the external one
				g.Expect(Kits(t, ctx, operatorNamespace)()).Should(WithTransform(integrationKitsToNamesTransform(), ContainElement(kit))) // the global one

				g.Expect(Lease(t, ctx, ns5, platform.OperatorLockName)()).To(BeNil(), "No locking Leases expected")
			})
		})

		g.Expect(Kamel(t, ctx, "uninstall", "-n", operatorNamespace, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())
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

func getKitImage(t *testing.T, ctx context.Context, ns string, name string) string {
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
	if err := TestClient(t).Get(ctx, key, &get); err != nil {
		return ""
	}
	return get.Status.Image
}
