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
	. "github.com/onsi/gomega"
	"testing"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/olm"
)

func TestBasicUninstall(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		Eventually(DefaultCamelCatalogPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))

		// should be completely removed on uninstall
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())

		// Roles only removed in non-olm use-case
		uninstallViaOLM := false
		var err error
		if uninstallViaOLM, err = olm.IsAPIAvailable(TestClient(t)); err != nil {
			t.Error(err)
			t.FailNow()
		}

		if !uninstallViaOLM {
			Eventually(Role(t, ns)).Should(BeNil())
			Eventually(RoleBinding(t, ns)).Should(BeNil())
			Eventually(ServiceAccount(t, ns, "camel-k-operator")).Should(BeNil())
		} else {
			Eventually(Role(t, ns)).ShouldNot(BeNil())
			Eventually(RoleBinding(t, ns)).ShouldNot(BeNil())
		}

		Eventually(Configmap(t, ns, "camel-k-maven-settings")).Should(BeNil())
		Eventually(OperatorPod(t, ns), TestTimeoutMedium).Should(BeNil())
		Eventually(KameletList(t, ns), TestTimeoutMedium).Should(BeEmpty())
		Eventually(CamelCatalogList(t, ns), TestTimeoutMedium).Should(BeEmpty())
	})
}

func TestUninstallSkipOperator(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except operator
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-operator").Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRole(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns, "--olm=false").Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except roles
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-roles").Execute()).To(Succeed())
		Eventually(Role(t, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRoleBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns, "--olm=false").Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except role-bindings
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-role-bindings").Execute()).To(Succeed())
		Eventually(RoleBinding(t, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipServiceAccounts(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns, "--olm=false").Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-service-accounts").Execute()).To(Succeed())
		Eventually(ServiceAccount(t, ns, "camel-k-operator")).ShouldNot(BeNil())
	})
}

func TestUninstallSkipIntegrationPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		// NOTE: skip CRDs is also required in addition to skip integration platform
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-integration-platform").Execute()).To(Succeed())
		Eventually(Platform(t, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipKamelets(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithIDAndKameletCatalog(t, operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		Eventually(KameletList(t, ns)).ShouldNot(BeEmpty())
		// on uninstall it should remove everything except kamelets
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-kamelets").Execute()).To(Succeed())
		Eventually(KameletList(t, ns)).ShouldNot(BeEmpty())
	})
}

func TestUninstallSkipCamelCatalogs(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		Eventually(CamelCatalogList(t, ns)).ShouldNot(BeEmpty())
		// on uninstall it should remove everything except camel catalogs
		Expect(Kamel(t, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-camel-catalogs").Execute()).To(Succeed())
		Eventually(CamelCatalogList(t, ns)).ShouldNot(BeEmpty())

	})
}
