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

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/pointer"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

/*
 * TODO
 * Test needs to be modified as taint test for java3 integration does not work on OCP.
 * Already skipped when executed using Kind since that is only a single-node cluster.
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestTolerationTrait(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("Run Java with node toleration operation exists", func(t *testing.T) {
			name := "java1"
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"--name", name,
				"-t", "toleration.enabled=true",
				"-t", "toleration.taints=camel.apache.org/master:NoExecute:300",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, name)()
			Expect(pod.Spec.Tolerations).NotTo(BeNil())

			Expect(pod.Spec.Tolerations).To(ContainElement(corev1.Toleration{
				Key:               "camel.apache.org/master",
				Operator:          corev1.TolerationOpExists,
				Effect:            corev1.TaintEffectNoExecute,
				TolerationSeconds: pointer.Int64(300),
			}))
		})

		t.Run("Run Java with node toleration operation equals", func(t *testing.T) {
			name := "java2"
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"--name", name,
				"-t", "toleration.enabled=true",
				"-t", "toleration.taints=camel.apache.org/master=test:NoExecute:300",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, name)()
			Expect(pod.Spec.Tolerations).NotTo(BeNil())

			Expect(pod.Spec.Tolerations).To(ContainElement(corev1.Toleration{
				Key:      "camel.apache.org/master",
				Operator: corev1.TolerationOpEqual,
				Value:    "test", Effect: corev1.TaintEffectNoExecute,
				TolerationSeconds: pointer.Int64(300),
			}))
		})

		t.Run("Run Java with master node toleration", func(t *testing.T) {
			if len(Nodes()()) == 1 {
				t.Skip("Skip master node toleration test on single-node cluster")
			}

			name := "java3"
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"--name", name,
				// Use the affinity trait to force the scheduling of the Integration pod onto a master node
				"-t", "affinity.enabled=true",
				"-t", "affinity.node-affinity-labels=node-role.kubernetes.io/master",
				// And tolerate the corresponding taint
				"-t", "toleration.enabled=true",
				"-t", "toleration.taints=node-role.kubernetes.io/master:NoSchedule",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, name)()
			Expect(pod).NotTo(BeNil())

			// Check the Integration pod contains the toleration
			Expect(pod.Spec.Tolerations).To(ContainElement(corev1.Toleration{
				Key:      "node-role.kubernetes.io/master",
				Operator: corev1.TolerationOpExists,
				Effect:   corev1.TaintEffectNoSchedule,
			}))

			// Check the Integration pod is running on a master node
			Expect(Node(pod.Spec.NodeName)()).NotTo(BeNil())
			Expect(Node(pod.Spec.NodeName)()).To(PointTo(MatchFields(IgnoreExtras, Fields{
				"Spec": MatchFields(IgnoreExtras, Fields{
					"Taints": ContainElement(corev1.Taint{
						Key:    "node-role.kubernetes.io/master",
						Effect: corev1.TaintEffectNoSchedule,
					}),
				}),
			})))
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
