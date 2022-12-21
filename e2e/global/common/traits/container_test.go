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

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestContainerTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-container"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("Container image pull policy and resources configuration", func(t *testing.T) {
			name := "java1"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"-t", "container.image-pull-policy=Always",
				"-t", "container.request-cpu=0.005",
				"-t", "container.request-memory=100Mi",
				"-t", "container.limit-cpu=200m",
				"-t", "container.limit-memory=500Mi",
				"--name", name,
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationPodHas(ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				imagePullPolicy := pod.Spec.Containers[0].ImagePullPolicy
				return imagePullPolicy == "Always"
			}), TestTimeoutShort).Should(BeTrue())
			Eventually(IntegrationPodHas(ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				limitsCpu := pod.Spec.Containers[0].Resources.Limits.Cpu()
				requestsCpu := pod.Spec.Containers[0].Resources.Requests.Cpu()
				return limitsCpu != nil && limitsCpu.String() == "200m" && requestsCpu != nil && requestsCpu.String() == "5m"
			}), TestTimeoutShort).Should(BeTrue())
			Eventually(IntegrationPodHas(ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				limitsMemory := pod.Spec.Containers[0].Resources.Limits.Memory()
				requestsMemory := pod.Spec.Containers[0].Resources.Requests.Memory()
				return limitsMemory != nil && limitsMemory.String() == "500Mi" && requestsMemory != nil && requestsMemory.String() == "100Mi"
			}), TestTimeoutShort).Should(BeTrue())

		})

		t.Run("Container name", func(t *testing.T) {
			name := "java2"
			containerName := "my-container-name"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"-t", "container.name="+containerName,
				"--name", name,
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationPodHas(ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				podContainerName := pod.Spec.Containers[0].Name
				return podContainerName == containerName
			}), TestTimeoutShort).Should(BeTrue())

		})

		// Clean-up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())

	})
}
