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
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestRunGlobalKamelet(t *testing.T) {
	forceGlobalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
	if !forceGlobalTest {
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)
		if ocp {
			t.Skip("Prefer not to run on OpenShift to avoid giving more permissions to the user running tests")
			return
		}
	}

	WithGlobalOperatorNamespace(t, func(operatorNamespace string) {
		operatorID := "camel-k-global-kamelet"
		Expect(KamelInstallWithID(operatorID, operatorNamespace, "--global", "--force").Execute()).To(Succeed())

		t.Run("Global operator + namespaced kamelet test", func(t *testing.T) {

			// NS2: namespace without operator
			WithNewTestNamespace(t, func(ns2 string) {
				Expect(CreateTimerKamelet(ns2, "my-own-timer-source")()).To(Succeed())

				Expect(KamelInstallWithID(operatorID, ns2, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())

				Expect(KamelRunWithID(operatorID, ns2, "files/timer-kamelet-usage.groovy").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns2, "timer-kamelet-usage"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns2, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
				Expect(Kamel("delete", "--all", "-n", ns2).Execute()).To(Succeed())
			})
		})

		t.Run("Global operator + global kamelet test", func(t *testing.T) {

			Expect(CreateTimerKamelet(operatorNamespace, "my-own-timer-source")()).To(Succeed())

			// NS3: namespace without operator
			WithNewTestNamespace(t, func(ns3 string) {
				Expect(KamelInstallWithID(operatorID, ns3, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())

				Expect(KamelRunWithID(operatorID, ns3, "files/timer-kamelet-usage.groovy").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns3, "timer-kamelet-usage"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns3, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
				Expect(Kamel("delete", "--all", "-n", ns3).Execute()).To(Succeed())
				Expect(TestClient().Delete(TestContext, Kamelet("my-own-timer-source", operatorNamespace)())).To(Succeed())
			})
		})

		Expect(Kamel("uninstall", "-n", operatorNamespace, "--skip-crd", "--skip-cluster-roles=false", "--skip-cluster-role-bindings=false").Execute()).To(Succeed())
	})

}
