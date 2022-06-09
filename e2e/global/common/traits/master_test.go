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

package traits

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
)

func TestMasterTrait(t *testing.T) {
	/*
	* TODO
	* The test just keeps randomly failing, either on kind or OCP4 clusters.
	* The integration times out before spinning up or tests for the Magicstring
	* fail.
	*
	* Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
	 */
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-master"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("master works", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/Master.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "master"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "master"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("only one integration with master runs", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/Master.java",
				"--name", "first",
				"--label", "leader-group=same",
				"-t", "master.label-key=leader-group",
				"-t", "master.label-value=same",
				"-t", "owner.target-labels=leader-group").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "first"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "first"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// Start a second integration with the same lock (it should not start the route)
			Expect(KamelRunWithID(operatorID, ns, "files/Master.java",
				"--name", "second",
				"--label", "leader-group=same",
				"-t", "master.label-key=leader-group",
				"-t", "master.label-value=same",
				"-t", "master.resource-name=first-lock",
				"-t", "owner.target-labels=leader-group").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "second"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "second"), TestTimeoutShort).Should(ContainSubstring("started in"))
			Eventually(IntegrationLogs(ns, "second"), 30*time.Second).ShouldNot(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
