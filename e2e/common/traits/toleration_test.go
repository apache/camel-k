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

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestTolerationTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		var wait int64 = 300

		InvokeUserTestCode(t, ns, func(ns string) {
			t.Run("Run Java with node toleration operation exists", func(t *testing.T) {
				Expect(Kamel("run", "-n", ns, "files/Java.java",
					"--name", "java1",
					"-t", "toleration.enabled=true",
					"-t", "toleration.taints=camel.apache.org/master:NoExecute:300").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns, "java1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
				Eventually(IntegrationCondition(ns, "java1", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
				Eventually(IntegrationLogs(ns, "java1"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

				pod := IntegrationPod(ns, "java1")()
				Expect(pod.Spec.Tolerations).NotTo(BeNil())

				Expect(pod.Spec.Tolerations).To(ContainElement(v1.Toleration{
					"camel.apache.org/master", v1.TolerationOpExists, "", v1.TaintEffectNoExecute, &wait,
				}))

				Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			})

			t.Run("Run Java with node toleration operation equals", func(t *testing.T) {
				Expect(Kamel("run", "-n", ns, "files/Java.java",
					"--name", "java2",
					"-t", "toleration.enabled=true",
					"-t", "toleration.taints=camel.apache.org/master=test:NoExecute:300").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns, "java2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
				Eventually(IntegrationCondition(ns, "java2", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
				Eventually(IntegrationLogs(ns, "java2"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

				pod := IntegrationPod(ns, "java2")()
				Expect(pod.Spec.Tolerations).NotTo(BeNil())

				Expect(pod.Spec.Tolerations).To(ContainElement(v1.Toleration{
					"camel.apache.org/master", v1.TolerationOpEqual, "test", v1.TaintEffectNoExecute, &wait,
				}))

				Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			})
		})
	})
}
