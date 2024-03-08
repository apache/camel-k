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
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {

		t.Run("master works", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Master.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, "master"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, "master"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("only one integration with master runs", func(t *testing.T) {
			nameFirst := RandomizedSuffixName("first")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Master.java",
				"--name", nameFirst,
				"--label", "leader-group=same",
				"-t", "master.label-key=leader-group",
				"-t", "master.label-value=same",
				"-t", "owner.target-labels=leader-group").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, nameFirst), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, nameFirst), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// Start a second integration with the same lock (it should not start the route)
			nameSecond := RandomizedSuffixName("second")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Master.java",
				"--name", nameSecond,
				"--label", "leader-group=same",
				"-t", "master.label-key=leader-group",
				"-t", "master.label-value=same",
				"-t", "master.resource-name=first-lock",
				"-t", "owner.target-labels=leader-group").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, nameSecond), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, nameSecond), TestTimeoutShort).Should(ContainSubstring("started in"))
			g.Eventually(IntegrationLogs(t, ns, nameSecond), 30*time.Second).ShouldNot(ContainSubstring("Magicstring!"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ns, nameFirst)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ns, nameFirst)()
			builderTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "addons", "master")
			g.Expect(builderTrait).ToNot(BeNil())
			g.Expect(len(builderTrait)).To(Equal(2))
			g.Expect(builderTrait["labelKey"]).To(Equal("leader-group"))
			g.Expect(builderTrait["labelValue"]).To(Equal("same"))
		})

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
