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

package advanced

import (
	"context"
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestCamelCatalogBuilder(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, g, ns)
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(PlatformConditionStatus(t, ctx, ns, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
		g.Eventually(CamelCatalog(t, ctx, ns, catalogName)).ShouldNot(BeNil())
		g.Eventually(CamelCatalogPhase(t, ctx, ns, catalogName), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))

		// Run an integration with a catalog not compatible
		// The operator should create the catalog, but fail on reconciliation as it is not compatible
		// and the integration should fail as well
		t.Run("Run catalog not compatible", func(t *testing.T) {
			name := RandomizedSuffixName("java-1-15")
			nonCompatibleCatalogName := "camel-catalog-1.15.0"
			g.Expect(
				KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "camel.runtime-version=1.15.0").Execute()).To(Succeed())

			g.Eventually(CamelCatalog(t, ctx, ns, nonCompatibleCatalogName)).ShouldNot(BeNil())
			g.Eventually(CamelCatalogPhase(t, ctx, ns, nonCompatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseError))
			g.Eventually(CamelCatalogCondition(t, ctx, ns, nonCompatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(ContainSubstring("Container image tool missing in catalog"))

			g.Eventually(IntegrationKit(t, ctx, ns, name)).ShouldNot(Equal(""))
			kitName := IntegrationKit(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, ns, kitName)).Should(Equal(v1.IntegrationKitPhaseError))
			g.Eventually(KitCondition(t, ctx, ns, kitName, v1.IntegrationKitConditionCatalogAvailable)().Reason).Should(Equal("Camel Catalog 1.15.0 error"))
			g.Eventually(IntegrationPhase(t, ctx, ns, name)).Should(Equal(v1.IntegrationPhaseError))
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionKitAvailable)().Status).Should(Equal(corev1.ConditionFalse))
		})

		// Run an integration with a compatible catalog
		// The operator should create the catalog, reconcile it properly and run the Integration accordingly
		t.Run("Run catalog compatible", func(t *testing.T) {
			name := RandomizedSuffixName("java-1-17")
			compatibleVersion := "1.17.0"
			compatibleCatalogName := "camel-catalog-" + strings.ToLower(compatibleVersion)

			// First of all we delete the catalog, if by any chance it was created previously
			g.Expect(DeleteCamelCatalog(t, ctx, ns, compatibleCatalogName)()).Should(BeTrue())
			g.Eventually(CamelCatalog(t, ctx, ns, compatibleCatalogName)).Should(BeNil())

			g.Expect(
				KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "camel.runtime-version="+compatibleVersion).Execute()).To(Succeed())

			g.Eventually(CamelCatalog(t, ctx, ns, compatibleCatalogName)).ShouldNot(BeNil())
			g.Eventually(CamelCatalogPhase(t, ctx, ns, compatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseReady))
			g.Eventually(CamelCatalogCondition(t, ctx, ns, compatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(Equal("Container image tool found in catalog"))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
		})

		t.Run("Run catalog container exists", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			compatibleVersion := "1.17.0"
			compatibleCatalogName := "camel-catalog-" + strings.ToLower(compatibleVersion)

			// First of all we delete the catalog, if by any chance it was created previously
			g.Expect(DeleteCamelCatalog(t, ctx, ns, compatibleCatalogName)()).Should(BeTrue())
			g.Eventually(CamelCatalog(t, ctx, ns, compatibleCatalogName)).Should(BeNil())

			g.Expect(
				KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "camel.runtime-version="+compatibleVersion).Execute()).To(Succeed())

			g.Eventually(CamelCatalog(t, ctx, ns, compatibleCatalogName)).ShouldNot(BeNil())
			g.Eventually(CamelCatalogPhase(t, ctx, ns, compatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseReady))
			g.Eventually(CamelCatalogCondition(t, ctx, ns, compatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(
				Equal("Container image tool found in catalog"),
			)

			g.Eventually(IntegrationKit(t, ctx, ns, name)).ShouldNot(Equal(""))
			kitName := IntegrationKit(t, ctx, ns, name)()
			g.Eventually(KitPhase(t, ctx, ns, kitName)).Should(Equal(v1.IntegrationKitPhaseReady))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})
	})
}
