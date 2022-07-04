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

package kustomize

import (
	"fmt"
	"os"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
)

func TestBasicSetup(t *testing.T) {
	if os.Getenv("CAMEL_K_CLUSTER_OCP3") == "true" {
		t.Skip("INFO: Skipping test as not supported on OCP3")
	}

	os.Setenv("MAKE_DIR", "../../../../install")

	// Ensure no CRDs are already installed
	UninstallAll()

	// Return the cluster to previous state
	defer Cleanup()

	WithNewTestNamespace(t, func(ns string) {
		ExecMake(t, Make("setup-cluster", fmt.Sprintf("NAMESPACE=%s", ns)))
		Eventually(CRDs()).Should(HaveLen(ExpCrds))

		ExecMake(t, Make("setup", fmt.Sprintf("NAMESPACE=%s", ns)))

		kroles := ExpKubePromoteRoles
		osroles := kroles + ExpOSPromoteRoles
		Eventually(Role(ns)).Should(Or(HaveLen(kroles), HaveLen(osroles)))

		kcroles := ExpKubeClusterRoles
		oscroles := kcroles + ExpOSClusterRoles
		Eventually(ClusterRole()).Should(Or(HaveLen(kcroles), HaveLen(oscroles)))

		// Tidy up to ensure next test works
		Expect(Kamel("uninstall", "-n", ns).Execute()).To(Succeed())
	})

}

func TestGlobalSetup(t *testing.T) {
	if os.Getenv("CAMEL_K_CLUSTER_OCP3") == "true" {
		t.Skip("INFO: Skipping test as not supported on OCP3")
	}

	os.Setenv("MAKE_DIR", "../../../../install")

	// Ensure no CRDs are already installed
	UninstallAll()

	// Return the cluster to previous state
	defer Cleanup()

	WithNewTestNamespace(t, func(ns string) {
		ExecMake(t, Make("setup-cluster", fmt.Sprintf("NAMESPACE=%s", ns)))
		Eventually(CRDs()).Should(HaveLen(ExpCrds))

		ExecMake(t, Make("setup", "GLOBAL=true", fmt.Sprintf("NAMESPACE=%s", ns)))

		Eventually(Role(ns)).Should(HaveLen(0))

		kcroles := ExpKubeClusterRoles + ExpKubePromoteRoles
		oscroles := kcroles + ExpOSClusterRoles + ExpOSPromoteRoles
		Eventually(ClusterRole()).Should(Or(HaveLen(kcroles), HaveLen(oscroles)))
	})
}
