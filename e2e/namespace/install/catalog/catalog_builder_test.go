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

package catalog

import (
	"fmt"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func TestCamelCatalogBuilder(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns, "--operator-env-vars", "KAMEL_INSTALL_DEFAULT_KAMELETS=false").Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(Platform(ns)).ShouldNot(BeNil())
		Eventually(PlatformConditionStatus(ns, v1.IntegrationPlatformConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
		Eventually(CamelCatalog(ns, catalogName)).ShouldNot(BeNil())
		catalog := CamelCatalog(ns, catalogName)()
		imageName := fmt.Sprintf("camel-k-runtime-%s-builder:%s", catalog.Spec.Runtime.Provider, strings.ToLower(catalog.Spec.Runtime.Version))
		Eventually(CamelCatalogPhase(ns, catalogName), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
		Eventually(CamelCatalogImage(ns, catalogName), TestTimeoutMedium).Should(ContainSubstring(imageName))
		// The container may have been created by previous test
		Eventually(CamelCatalogCondition(ns, catalogName, v1.CamelCatalogConditionReady)().Message).Should(
			Or(Equal("Container image successfully built"), Equal("Container image exists on registry")),
		)

		// Run an integration with a catalog not compatible
		// The operator should create the catalog, but fail on reconciliation and the integration should fail as well
		t.Run("Run catalog not compatible", func(t *testing.T) {
			name := "java"
			nonCompatibleCatalogName := "camel-catalog-1.15.0-quarkus"
			Expect(
				KamelRunWithID(operatorID, ns, "../files/Java.java", "--name", name,
					"-t", "camel.runtime-version=1.15.0",
				).Execute()).To(Succeed())

			Eventually(CamelCatalog(ns, nonCompatibleCatalogName)).ShouldNot(BeNil())
			Eventually(CamelCatalogPhase(ns, nonCompatibleCatalogName)).Should(Equal(v1.CamelCatalogPhaseError))
			Eventually(CamelCatalogCondition(ns, nonCompatibleCatalogName, v1.CamelCatalogConditionReady)().Message).Should(ContainSubstring("Missing base image"))

			Eventually(IntegrationKit(ns, name)).ShouldNot(Equal(""))
			kitName := IntegrationKit(ns, name)()
			Eventually(KitPhase(ns, kitName)).Should(Equal(v1.IntegrationKitPhaseError))
			Eventually(KitCondition(ns, kitName, v1.IntegrationKitConditionCatalogAvailable)().Reason).Should(Equal("Camel Catalog 1.15.0 error"))
			Eventually(IntegrationPhase(ns, name)).Should(Equal(v1.IntegrationPhaseError))
			Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionKitAvailable)().Status).Should(Equal(corev1.ConditionFalse))

			// Clean up
			Eventually(DeleteIntegrations(ns), TestTimeoutLong).Should(Equal(0))
		})
	})

	WithNewTestNamespace(t, func(ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns, "--operator-env-vars", "KAMEL_INSTALL_DEFAULT_KAMELETS=false").Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(Platform(ns)).ShouldNot(BeNil())
		Eventually(PlatformConditionStatus(ns, v1.IntegrationPlatformConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		catalogName := fmt.Sprintf("camel-catalog-%s", strings.ToLower(defaults.DefaultRuntimeVersion))
		Eventually(CamelCatalog(ns, catalogName)).ShouldNot(BeNil())
		catalog := CamelCatalog(ns, catalogName)()
		imageName := fmt.Sprintf("camel-k-runtime-%s-builder:%s", catalog.Spec.Runtime.Provider, strings.ToLower(catalog.Spec.Runtime.Version))
		Eventually(CamelCatalogPhase(ns, catalogName), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
		Eventually(CamelCatalogImage(ns, catalogName), TestTimeoutMedium).Should(ContainSubstring(imageName))
		Eventually(CamelCatalogCondition(ns, catalogName, v1.CamelCatalogConditionReady)().Message).Should(Equal("Container image exists on registry"))
	})
}
