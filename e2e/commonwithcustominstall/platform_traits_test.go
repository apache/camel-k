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

package commonwithcustominstall

import (
	"testing"

	"github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestTraitOnIntegrationPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-platform-trait-test"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		containerTestName := "testname"
		ip := Platform(ns)()
		ip.Spec.Traits = v1.Traits{Logging: &trait.LoggingTrait{Level: "DEBUG"}, Container: &trait.ContainerTrait{Name: containerTestName}}

		if err := TestClient().Update(TestContext, ip); err != nil {
			t.Fatal("Can't create IntegrationPlatform", err)
		}

		name := RandomizedSuffixName("java")
		t.Run("Run integration with platform traits", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"--name", name,
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Expect(IntegrationPod(ns, name)().Spec.Containers[0].Name).To(BeEquivalentTo(containerTestName))

			found := false
			for _, env := range IntegrationPod(ns, name)().Spec.Containers[0].Env {
				if env.Name == "QUARKUS_LOG_LEVEL" {
					Expect(env.Value).To(BeEquivalentTo("DEBUG"))
					found = true
					break
				}
			}
			Expect(found).To(BeTrue(), "Can't find QUARKUS_LOG_LEVEL ENV variable")
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("DEBUG"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
