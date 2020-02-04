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

package e2e

import (
	"testing"

	. "github.com/onsi/gomega"
)

func TestBasicUninstall(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// should be completely removed on uninstall
		Expect(kamel("uninstall", "-n", ns).Execute()).Should(BeNil())
		Eventually(role(ns)).Should(BeNil())
		Eventually(rolebinding(ns)).Should(BeNil())
		Eventually(configmap(ns,"camel-k-maven-settings")).Should(BeNil())
		Eventually(clusterrole(ns)).Should(BeNil())
		Eventually(serviceaccount(ns,"camel-k-maven-settings")).Should(BeNil())
		Eventually(operatorPod(ns)).Should(BeNil())
	})
}

func TestUninstallSkipOperator(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except operator
		Expect(kamel("uninstall", "-n", ns,"--skip-operator").Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRole(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except roles
		Expect(kamel("uninstall", "-n", ns,"--skip-roles").Execute()).Should(BeNil())
		Eventually(role(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipRoleBinding(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except role-bindings
		Expect(kamel("uninstall", "-n", ns,"--skip-role-bindings").Execute()).Should(BeNil())
		Eventually(rolebinding(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipClusterRoles(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		Expect(kamel("uninstall", "-n", ns,"--skip-cluster-roles").Execute()).Should(BeNil())
		Eventually(clusterrole(ns)).ShouldNot(BeNil())
	})
}

func TestUninstallSkipServiceAccounts(t *testing.T) {
	//t.Skip("inconsistent test results ")
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		Expect(kamel("uninstall", "-n", ns,"--skip-service-accounts").Execute()).Should(BeNil())
		Eventually(serviceaccount(ns, "camel-k-operator")).ShouldNot(BeNil())
	})
}

func TestUninstallSkipIntegrationPlatform(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		// a successful new installation
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		// on uninstall it should remove everything except cluster-roles
		// NOTE: skip CRDs is also required in addition to skip integration platform
		Expect(kamel("uninstall", "-n", ns,"--skip-crd","--skip-integration-platform").Execute()).Should(BeNil())
		Eventually(platform(ns)).ShouldNot(BeNil())
	})
}