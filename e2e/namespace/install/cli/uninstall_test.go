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

package common

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

func TestBasicUninstall(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())

		// should be completely removed on uninstall
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())
		Eventually(Role(ns)).Should(BeNil())
		Eventually(RoleBinding(ns)).Should(BeNil())
		Eventually(Configmap(ns, "camel-k-maven-settings")).Should(BeNil())
		Eventually(ServiceAccount(ns, "camel-k-operator")).Should(BeNil())
		Eventually(OperatorPod(ns)).Should(BeNil())
		Eventually(KameletList(ns)).Should(BeEmpty())
	})
}

func TestUninstallSkipOperator(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except operator
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-operator").Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRole(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns, "--olm=false").Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except roles
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-roles").Execute()).To(Succeed())
		Eventually(Role(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRoleBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns, "--olm=false").Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except role-bindings
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-role-bindings").Execute()).To(Succeed())
		Eventually(RoleBinding(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipServiceAccounts(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns, "--olm=false").Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-service-accounts").Execute()).To(Succeed())
		Eventually(ServiceAccount(ns, "camel-k-operator")).ShouldNot(BeNil())
	})
}

func TestUninstallSkipIntegrationPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		// NOTE: skip CRDs is also required in addition to skip integration platform
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-integration-platform").Execute()).To(Succeed())
		Eventually(Platform(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipKamelets(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(KameletList(ns)).ShouldNot(BeEmpty())
		// on uninstall it should remove everything except kamelets
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-kamelets").Execute()).To(Succeed())
		Eventually(KameletList(ns)).ShouldNot(BeEmpty())
	})
}
