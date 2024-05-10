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
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/olm"
)

func TestBasicUninstall(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(DefaultCamelCatalogPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))

		// should be completely removed on uninstall
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())

		// Roles only removed in non-olm use-case
		uninstallViaOLM := false
		var err error
		if uninstallViaOLM, err = olm.IsAPIAvailable(TestClient(t)); err != nil {
			t.Error(err)
			t.FailNow()
		}

		if !uninstallViaOLM {
			g.Eventually(Role(t, ctx, ns)).Should(BeNil())
			g.Eventually(RoleBinding(t, ctx, ns)).Should(BeNil())
			g.Eventually(ServiceAccount(t, ctx, ns, "camel-k-operator")).Should(BeNil())
		} else {
			g.Eventually(Role(t, ctx, ns)).ShouldNot(BeNil())
			g.Eventually(RoleBinding(t, ctx, ns)).ShouldNot(BeNil())
		}

		g.Eventually(Configmap(t, ctx, ns, "camel-k-maven-settings")).Should(BeNil())
		g.Eventually(OperatorPod(t, ctx, ns), TestTimeoutMedium).Should(BeNil())
		g.Eventually(KameletList(t, ctx, ns), TestTimeoutMedium).Should(BeEmpty())
		g.Eventually(CamelCatalogList(t, ctx, ns), TestTimeoutMedium).Should(BeEmpty())
	})
}

func TestUninstallSkipOperator(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except operator
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-operator").Execute()).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRole(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--olm=false")).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except roles
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-roles").Execute()).To(Succeed())
		g.Eventually(Role(t, ctx, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRoleBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--olm=false")).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except role-bindings
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-role-bindings").Execute()).To(Succeed())
		g.Eventually(RoleBinding(t, ctx, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipServiceAccounts(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--olm=false")).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-service-accounts").Execute()).To(Succeed())
		g.Eventually(ServiceAccount(t, ctx, ns, "camel-k-operator")).ShouldNot(BeNil())
	})
}

func TestUninstallSkipIntegrationPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		// NOTE: skip CRDs is also required in addition to skip integration platform
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-integration-platform").Execute()).To(Succeed())
		g.Eventually(Platform(t, ctx, ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipKamelets(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithIDAndKameletCatalog(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(KameletList(t, ctx, ns)).ShouldNot(BeEmpty())
		// on uninstall it should remove everything except kamelets
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-kamelets").Execute()).To(Succeed())
		g.Eventually(KameletList(t, ctx, ns)).ShouldNot(BeEmpty())
	})
}

func TestUninstallSkipCamelCatalogs(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(CamelCatalogList(t, ctx, ns)).ShouldNot(BeEmpty())
		// on uninstall it should remove everything except camel catalogs
		g.Expect(Kamel(t, ctx, "uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-camel-catalogs").Execute()).To(Succeed())
		g.Eventually(CamelCatalogList(t, ctx, ns)).ShouldNot(BeEmpty())

	})
}
