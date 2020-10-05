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
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
)

func TestBasicUninstall(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())

		// should be completely removed on uninstall
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute()).Should(BeNil())
		Eventually(Role(ns)).Should(BeNil())
		Eventually(Rolebinding(ns)).Should(BeNil())
		Eventually(Configmap(ns, "camel-k-maven-settings")).Should(BeNil())
		Eventually(ServiceAccount(ns, "camel-k-operator")).Should(BeNil())
		Eventually(OperatorPod(ns)).Should(BeNil())
	})
}

func TestUninstallSkipOperator(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except operator
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-operator").Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRole(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except roles
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-roles").Execute()).Should(BeNil())
		Eventually(Role(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRoleBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except role-bindings
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-role-bindings").Execute()).Should(BeNil())
		Eventually(Rolebinding(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipServiceAccounts(t *testing.T) {
	//t.Skip("inconsistent test results ")
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-service-accounts").Execute()).Should(BeNil())
		Eventually(ServiceAccount(ns, "camel-k-operator")).ShouldNot(BeNil())
	})
}

func TestUninstallSkipIntegrationPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		// NOTE: skip CRDs is also required in addition to skip integration platform
		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles", "--skip-integration-platform").Execute()).Should(BeNil())
		Eventually(Platform(ns)).ShouldNot(BeNil())
	})
}
