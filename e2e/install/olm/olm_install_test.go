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

package olm

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/e2e/support/util"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

const installCatalogSourceName = "test-camel-k-source"

func TestOLMInstallationOwnNamespace(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
		// Let's make sure no CRD is yet available in the cluster
		// as we must make the procedure to install them accordingly
		g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
		bundleImageName, ok := os.LookupEnv("BUNDLE_IMAGE_NAME")
		g.Expect(ok).To(BeTrue(), "Missing bundle image: you need to build and push to a container registry and set BUNDLE_IMAGE_NAME env var")
		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
		// Install staged bundle (it must be available by building it before running the test)
		// You can build it locally via `make bundle-push` action
		ExpectExecSucceedWithTimeout(t, g,
			Make(t,
				"bundle-test",
				fmt.Sprintf("BUNDLE_IMAGE_NAME=%s", bundleImageName),
				fmt.Sprintf("NAMESPACE=%s", operatorNs),
				fmt.Sprintf("OLM_INSTALL_MODE=%s", "OwnNamespace"),
			),
			"300s",
		)
		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)
		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, operatorNs), TestTimeoutMedium).
			Should(Equal(olm.CSVPhaseSucceeded))
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, operatorNs), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(OperatorImage(t, ctx, operatorNs)).Should(Equal(operatorImage()))

		// Check if restricted security context has been applyed
		t.Run("check operator security context", func(t *testing.T) {
			operatorPod := OperatorPod(t, ctx, operatorNs)()
			g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(DefaultOperatorSecurityContext().RunAsNonRoot))
			g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(DefaultOperatorSecurityContext().Capabilities))
			g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(DefaultOperatorSecurityContext().SeccompProfile))
			g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(DefaultOperatorSecurityContext().AllowPrivilegeEscalation))
		})

		t.Run("run integration in own namespace", func(t *testing.T) {
			// Test a simple integration is running
			g.Expect(KamelRun(t, ctx, operatorNs, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, operatorNs, "yaml", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, operatorNs, "yaml")).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, operatorNs, "yaml")).Should(ContainSubstring("Magicstring!"))
		})

		// NOTE: this is required for debugging purposes only
		util.Dump(ctx, TestClient(t), operatorNs, t)

		t.Run("run integration another namespace", func(t *testing.T) {
			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, anotherNs string) {
				// The operator has not to reconcile it
				g.Expect(KamelRun(t, ctx, anotherNs, "files/yaml.yaml").Execute()).To(Succeed())
				g.Consistently(IntegrationPhase(t, ctx, anotherNs, "yaml"), 30*time.Second, 5*time.Second).
					Should(BeEmpty())
			})
		})

		t.Run("delete operator", func(t *testing.T) {
			// Remove OLM CSV and test Integration is still existing
			csv := ClusterServiceVersion(t, ctx, noAdditionalConditions, operatorNs)()
			g.Expect(TestClient(t).Delete(ctx, csv)).To(Succeed())
			g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())

			g.Consistently(Integration(t, ctx, operatorNs, "yaml"), 15*time.Second, 5*time.Second).ShouldNot(BeNil())
			g.Consistently(
				IntegrationConditionStatus(t, ctx, operatorNs, "yaml", v1.IntegrationConditionReady), 15*time.Second, 5*time.Second).
				Should(Equal(corev1.ConditionTrue))

			// Test CRD uninstall (will remove Integrations as well)
			UninstallCRDs(t, ctx, g, "../../../")
			g.Eventually(CRDs(t)).Should(BeNil())
		})
	})
}

// func TestOLMInstallationAllNamespaces(t *testing.T) {
// 	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
// 		// Let's make sure no CRD is yet available in the cluster
// 		// as we must make the procedure to install them accordingly
// 		g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
// 		bundleImageName, ok := os.LookupEnv("BUNDLE_IMAGE_NAME")
// 		g.Expect(ok).To(BeTrue(), "Missing bundle image: you need to build and push to a container registry and set BUNDLE_IMAGE_NAME env var")
// 		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
// 		// Install staged bundle (it must be available by building it before running the test)
// 		// You can build it locally via `make bundle-push` action
// 		ExpectExecSucceedWithTimeout(t, g,
// 			Make(t,
// 				"bundle-test",
// 				fmt.Sprintf("BUNDLE_IMAGE_NAME=%s", bundleImageName),
// 				fmt.Sprintf("NAMESPACE=%s", operatorNs),
// 				fmt.Sprintf("OLM_INSTALL_MODE=%s", "AllNamespaces"),
// 			),
// 			"300s",
// 		)
// 		// Refresh the test client to account for the newly installed CRDs
// 		RefreshClient(t)
// 		// Find the only one Camel K CSV
// 		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
// 			return true
// 		}
// 		g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, operatorNs), TestTimeoutMedium).
// 			Should(Equal(olm.CSVPhaseSucceeded))
// 		// Check the operator pod is running
// 		g.Eventually(OperatorPodPhase(t, ctx, operatorNs), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
// 		g.Eventually(OperatorImage(t, ctx, operatorNs)).Should(Equal(operatorImage()))

// 		t.Run("run integration in any namespace", func(t *testing.T) {
// 			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, anyNs string) {
// 				g.Expect(KamelRun(t, ctx, anyNs, "files/yaml.yaml").Execute()).To(Succeed())
// 				g.Eventually(IntegrationConditionStatus(t, ctx, anyNs, "yaml", v1.IntegrationConditionReady), TestTimeoutMedium).
// 					Should(Equal(corev1.ConditionTrue))
// 				g.Eventually(IntegrationPodPhase(t, ctx, anyNs, "yaml")).Should(Equal(corev1.PodRunning))
// 				g.Eventually(IntegrationLogs(t, ctx, anyNs, "yaml")).Should(ContainSubstring("Magicstring!"))

// 				t.Run("delete operator", func(t *testing.T) {
// 					// Remove OLM CSV and test Integration is still existing
// 					csv := ClusterServiceVersion(t, ctx, noAdditionalConditions, operatorNs)()
// 					g.Expect(TestClient(t).Delete(ctx, csv)).To(Succeed())
// 					g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())

// 					g.Consistently(Integration(t, ctx, anyNs, "yaml"), 15*time.Second, 5*time.Second).ShouldNot(BeNil())
// 					g.Consistently(
// 						IntegrationConditionStatus(t, ctx, anyNs, "yaml", v1.IntegrationConditionReady), 15*time.Second, 5*time.Second).
// 						Should(Equal(corev1.ConditionTrue))

// 					// Test CRD uninstall (will remove Integrations as well)
// 					UninstallCRDs(t, ctx, g, "../../../")
// 					g.Eventually(CRDs(t)).Should(BeNil())
// 				})
// 			})
// 		})
// 	})
// }

// func TestOLMInstallationSingleNamespace(t *testing.T) {
// 	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
// 		// Let's make sure no CRD is yet available in the cluster
// 		// as we must make the procedure to install them accordingly
// 		g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
// 		bundleImageName, ok := os.LookupEnv("BUNDLE_IMAGE_NAME")
// 		g.Expect(ok).To(BeTrue(), "Missing bundle image: you need to build and push to a container registry and set BUNDLE_IMAGE_NAME env var")
// 		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
// 		// Install staged bundle (it must be available by building it before running the test)
// 		// You can build it locally via `make bundle-push` action

// 		// Targeted namespace
// 		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, targetNs string) {
// 			ExpectExecSucceedWithTimeout(t, g,
// 				Make(t,
// 					"bundle-test",
// 					fmt.Sprintf("BUNDLE_IMAGE_NAME=%s", bundleImageName),
// 					fmt.Sprintf("NAMESPACE=%s", operatorNs),
// 					fmt.Sprintf("OLM_INSTALL_MODE=%s=%s", "SingleNamespace", targetNs),
// 				),
// 				"300s",
// 			)
// 			// Refresh the test client to account for the newly installed CRDs
// 			RefreshClient(t)
// 			// Find the only one Camel K CSV
// 			noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
// 				return true
// 			}
// 			g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, operatorNs), TestTimeoutMedium).
// 				Should(Equal(olm.CSVPhaseSucceeded))
// 			// Check the operator pod is running
// 			g.Eventually(OperatorPodPhase(t, ctx, operatorNs), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
// 			g.Eventually(OperatorImage(t, ctx, operatorNs)).Should(Equal(operatorImage()))

// 			t.Run("run integration in target namespace", func(t *testing.T) {
// 				g.Expect(KamelRun(t, ctx, targetNs, "files/yaml.yaml").Execute()).To(Succeed())
// 				g.Eventually(IntegrationConditionStatus(t, ctx, targetNs, "yaml", v1.IntegrationConditionReady), TestTimeoutMedium).
// 					Should(Equal(corev1.ConditionTrue))
// 				g.Eventually(IntegrationPodPhase(t, ctx, targetNs, "yaml")).Should(Equal(corev1.PodRunning))
// 				g.Eventually(IntegrationLogs(t, ctx, targetNs, "yaml")).Should(ContainSubstring("Magicstring!"))
// 			})

// 			t.Run("run integration in another namespace", func(t *testing.T) {
// 				WithNewTestNamespace(t, func(ctx context.Context, g *WithT, anotherNs string) {
// 					// The operator has not to reconcile it
// 					g.Expect(KamelRun(t, ctx, anotherNs, "files/yaml.yaml").Execute()).To(Succeed())
// 					g.Consistently(IntegrationPhase(t, ctx, anotherNs, "yaml"), 30*time.Second, 5*time.Second).
// 						Should(BeEmpty())
// 				})
// 			})

// 			t.Run("delete operator", func(t *testing.T) {
// 				// Remove OLM CSV and test Integration is still existing
// 				csv := ClusterServiceVersion(t, ctx, noAdditionalConditions, operatorNs)()
// 				g.Expect(TestClient(t).Delete(ctx, csv)).To(Succeed())
// 				g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())

// 				g.Consistently(Integration(t, ctx, targetNs, "yaml"), 15*time.Second, 5*time.Second).ShouldNot(BeNil())
// 				g.Consistently(
// 					IntegrationConditionStatus(t, ctx, targetNs, "yaml", v1.IntegrationConditionReady), 15*time.Second, 5*time.Second).
// 					Should(Equal(corev1.ConditionTrue))

// 				// Test CRD uninstall (will remove Integrations as well)
// 				UninstallCRDs(t, ctx, g, "../../../")
// 				g.Eventually(CRDs(t)).Should(BeNil())
// 			})
// 		})
// 	})
// }

// func TestOLMInstallationMultiNamespaces(t *testing.T) {
// 	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
// 		// Let's make sure no CRD is yet available in the cluster
// 		// as we must make the procedure to install them accordingly
// 		g.Eventually(CRDs(t)).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
// 		bundleImageName, ok := os.LookupEnv("BUNDLE_IMAGE_NAME")
// 		g.Expect(ok).To(BeTrue(), "Missing bundle image: you need to build and push to a container registry and set BUNDLE_IMAGE_NAME env var")
// 		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
// 		// Install staged bundle (it must be available by building it before running the test)
// 		// You can build it locally via `make bundle-push` action

// 		// Target 1 NS
// 		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns1 string) {
// 			// Target 2 NS
// 			WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns2 string) {
// 				ExpectExecSucceedWithTimeout(t, g,
// 					Make(t,
// 						"bundle-test",
// 						fmt.Sprintf("BUNDLE_IMAGE_NAME=%s", bundleImageName),
// 						fmt.Sprintf("NAMESPACE=%s", operatorNs),
// 						fmt.Sprintf("OLM_INSTALL_MODE=%s=%s,%s", "MultiNamespace", ns1, ns2),
// 					),
// 					"300s",
// 				)
// 				// Refresh the test client to account for the newly installed CRDs
// 				RefreshClient(t)
// 				// Find the only one Camel K CSV
// 				noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
// 					return true
// 				}
// 				g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, operatorNs), TestTimeoutMedium).
// 					Should(Equal(olm.CSVPhaseSucceeded))
// 				// Check the operator pod is running
// 				g.Eventually(OperatorPodPhase(t, ctx, operatorNs), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
// 				g.Eventually(OperatorImage(t, ctx, operatorNs)).Should(Equal(operatorImage()))

// 				t.Run("run integration in target 1 namespace", func(t *testing.T) {
// 					g.Expect(KamelRun(t, ctx, ns1, "files/yaml.yaml").Execute()).To(Succeed())
// 					g.Eventually(IntegrationConditionStatus(t, ctx, ns1, "yaml", v1.IntegrationConditionReady), TestTimeoutMedium).
// 						Should(Equal(corev1.ConditionTrue))
// 					g.Eventually(IntegrationPodPhase(t, ctx, ns1, "yaml")).Should(Equal(corev1.PodRunning))
// 					g.Eventually(IntegrationLogs(t, ctx, ns1, "yaml")).Should(ContainSubstring("Magicstring!"))
// 				})

// 				t.Run("run integration in target 2 namespace", func(t *testing.T) {
// 					g.Expect(KamelRun(t, ctx, ns2, "files/yaml.yaml").Execute()).To(Succeed())
// 					g.Eventually(IntegrationConditionStatus(t, ctx, ns2, "yaml", v1.IntegrationConditionReady), TestTimeoutMedium).
// 						Should(Equal(corev1.ConditionTrue))
// 					g.Eventually(IntegrationPodPhase(t, ctx, ns2, "yaml")).Should(Equal(corev1.PodRunning))
// 					g.Eventually(IntegrationLogs(t, ctx, ns2, "yaml")).Should(ContainSubstring("Magicstring!"))
// 				})

// 				t.Run("run integration in another namespace (untargeted)", func(t *testing.T) {
// 					WithNewTestNamespace(t, func(ctx context.Context, g *WithT, anotherNs string) {
// 						// The operator has not to reconcile it
// 						g.Expect(KamelRun(t, ctx, anotherNs, "files/yaml.yaml").Execute()).To(Succeed())
// 						g.Consistently(IntegrationPhase(t, ctx, anotherNs, "yaml"), 30*time.Second, 5*time.Second).
// 							Should(BeEmpty())
// 					})
// 				})

// 				t.Run("delete operator", func(t *testing.T) {
// 					// Remove OLM CSV and test Integration is still existing
// 					csv := ClusterServiceVersion(t, ctx, noAdditionalConditions, operatorNs)()
// 					g.Expect(TestClient(t).Delete(ctx, csv)).To(Succeed())
// 					g.Eventually(OperatorPod(t, ctx, operatorNs)).Should(BeNil())

// 					g.Consistently(Integration(t, ctx, ns1, "yaml"), 15*time.Second, 5*time.Second).ShouldNot(BeNil())
// 					g.Consistently(
// 						IntegrationConditionStatus(t, ctx, ns1, "yaml", v1.IntegrationConditionReady), 15*time.Second, 5*time.Second).
// 						Should(Equal(corev1.ConditionTrue))

// 					// Test CRD uninstall (will remove Integrations as well)
// 					UninstallCRDs(t, ctx, g, "../../../")
// 					g.Eventually(CRDs(t)).Should(BeNil())
// 				})
// 			})
// 		})
// 	})
// }

func operatorImage() string {
	return fmt.Sprintf("%s:%s", defaults.ImageName, defaults.Version)
}
