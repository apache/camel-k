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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestMasterTrait(t *testing.T) {
	RegisterTestingT(t)

	t.Run("master works", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Master.java").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "master"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, "master"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("only one integration with master runs", func(t *testing.T) {
		nameFirst := RandomizedSuffixName("first")
		Expect(KamelRunWithID(operatorID, ns, "files/Master.java",
			"--name", nameFirst,
			"--label", "leader-group=same",
			"-t", "master.label-key=leader-group",
			"-t", "master.label-value=same",
			"-t", "owner.target-labels=leader-group").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, nameFirst), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, nameFirst), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		// Start a second integration with the same lock (it should not start the route)
		nameSecond := RandomizedSuffixName("second")
		Expect(KamelRunWithID(operatorID, ns, "files/Master.java",
			"--name", nameSecond,
			"--label", "leader-group=same",
			"-t", "master.label-key=leader-group",
			"-t", "master.label-value=same",
			"-t", "master.resource-name=first-lock",
			"-t", "owner.target-labels=leader-group").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, nameSecond), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, nameSecond), TestTimeoutShort).Should(ContainSubstring("started in"))
		Eventually(IntegrationLogs(ns, nameSecond), 30*time.Second).ShouldNot(ContainSubstring("Magicstring!"))

		// check integration schema does not contains unwanted default trait value.
		Eventually(UnstructuredIntegration(ns, nameFirst)).ShouldNot(BeNil())
		unstructuredIntegration := UnstructuredIntegration(ns, nameFirst)()
		builderTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "addons", "master")
		Expect(builderTrait).ToNot(BeNil())
		Expect(len(builderTrait)).To(Equal(2))
		Expect(builderTrait["labelKey"]).To(Equal("leader-group"))
		Expect(builderTrait["labelValue"]).To(Equal("same"))
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
