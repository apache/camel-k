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

package commonwithcustominstall

import (
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

	WithNewTestNamespace(t, func(ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(Platform(ns)).ShouldNot(BeNil())
		Eventually(PlatformConditionStatus(ns, v1.IntegrationPlatformConditionTypeCreated), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
		Eventually(CamelCatalog(ns, catalogName)).ShouldNot(BeNil())
		Eventually(CamelCatalogPhase(ns, catalogName), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))

		// Run an integration with a catalog not compatible
		// The operator should create the catalog, but fail on reconciliation as it is not compatible
		// and the integration should fail as well
		t.Run("Run catalog not compatible", func(t *testing.T) {
			name := RandomizedSuffixName("java-1-15")
			nonCompatibleCatalogName := "camel-catalog-1.15.0"
			Expect(
				KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
					"-t", "camel.runtime-version=1.15.0",
				).Execute()).To(Succeed())

			Eventually(CamelCatalog(ns, nonCompatibleCatalogName)).ShouldNot(BeNil())
			Eventually(CamelCatalogPhase(ns, nonCompatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseError))
			Eventually(CamelCatalogCondition(ns, nonCompatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(ContainSubstring("Container image tool missing in catalog"))

			Eventually(IntegrationKit(ns, name)).ShouldNot(Equal(""))
			kitName := IntegrationKit(ns, name)()
			Eventually(KitPhase(ns, kitName)).Should(Equal(v1.IntegrationKitPhaseError))
			Eventually(KitCondition(ns, kitName, v1.IntegrationKitConditionCatalogAvailable)().Reason).Should(Equal("Camel Catalog 1.15.0 error"))
			Eventually(IntegrationPhase(ns, name)).Should(Equal(v1.IntegrationPhaseError))
			Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionKitAvailable)().Status).Should(Equal(corev1.ConditionFalse))

			// Clean up
			Eventually(DeleteIntegrations(ns)).Should(Equal(0))
		})

		// Run an integration with a compatible catalog
		// The operator should create the catalog, reconcile it properly and run the Integration accordingly
		t.Run("Run catalog compatible", func(t *testing.T) {
			name := RandomizedSuffixName("java-1-17")
			compatibleVersion := "1.17.0"
			compatibleCatalogName := "camel-catalog-" + strings.ToLower(compatibleVersion)

			// First of all we delete the catalog, if by any chance it was created previously
			Expect(DeleteCamelCatalog(ns, compatibleCatalogName)()).Should(BeTrue())
			Eventually(CamelCatalog(ns, compatibleCatalogName)).Should(BeNil())

			Expect(
				KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
					"-t", "camel.runtime-version="+compatibleVersion,
				).Execute()).To(Succeed())

			Eventually(CamelCatalog(ns, compatibleCatalogName)).ShouldNot(BeNil())
			Eventually(CamelCatalogPhase(ns, compatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseReady))
			Eventually(CamelCatalogCondition(ns, compatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(Equal("Container image tool found in catalog"))
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))

			// Clean up
			Eventually(DeleteIntegrations(ns)).Should(Equal(0))
		})

		t.Run("Run catalog container exists", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			compatibleVersion := "1.17.0"
			compatibleCatalogName := "camel-catalog-" + strings.ToLower(compatibleVersion)

			// First of all we delete the catalog, if by any chance it was created previously
			Expect(DeleteCamelCatalog(ns, compatibleCatalogName)()).Should(BeTrue())
			Eventually(CamelCatalog(ns, compatibleCatalogName)).Should(BeNil())

			Expect(
				KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
					"-t", "camel.runtime-version="+compatibleVersion,
				).Execute()).To(Succeed())

			Eventually(CamelCatalog(ns, compatibleCatalogName)).ShouldNot(BeNil())
			Eventually(CamelCatalogPhase(ns, compatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseReady))
			Eventually(CamelCatalogCondition(ns, compatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(
				Equal("Container image tool found in catalog"),
			)

			Eventually(IntegrationKit(ns, name)).ShouldNot(Equal(""))
			kitName := IntegrationKit(ns, name)()
			Eventually(KitPhase(ns, kitName)).Should(Equal(v1.IntegrationKitPhaseReady))
			Eventually(IntegrationPodPhase(ns, name)).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Clean up
			Eventually(DeleteIntegrations(ns)).Should(Equal(0))
		})

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
