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
	"os"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

// TestTektonLikeBehavior verifies that the kamel binary can be invoked from within the Camel K image.
// This feature is used in Tekton pipelines.

/*
 * TODO
 * Test has issues on OCP4.
 * Since that cluster installs via OLM, permissions are required for the
 * service account doing the install to access the OLM, eg. get:subscriptions.
 *
 * Need to change the service account used to execute this test as do not
 * want to extend the permissions of the operator service account
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestTektonLikeBehavior(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(CreateOperatorServiceAccount(ns)).To(Succeed())
		Expect(CreateOperatorRole(ns)).To(Succeed())
		Expect(CreateOperatorRoleBinding(ns)).To(Succeed())

		Eventually(OperatorPod(ns)).Should(BeNil())
		Expect(CreateKamelPod(ns, "tekton-task", "install", "--skip-cluster-setup", "--force")).To(Succeed())

		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	})
}
