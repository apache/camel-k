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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// This test requires operator installation with a custom operator ID, thus needs
// to be run under e2e/namespace.
func TestKameletFromCustomRepository(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		kameletName := "timer-custom-source"
		removeKamelet(kameletName, ns)

		Eventually(Kamelet(kameletName, ns)).Should(BeNil())

		// Add the custom repository
		Expect(Kamel("kamelet", "add-repo",
			"github:apache/camel-k/e2e/global/common/files/kamelets",
			"-n", ns,
			"-x", operatorID).Execute()).To(Succeed())

		Expect(KamelRunWithID(operatorID, ns, "files/TimerCustomKameletIntegration.java").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-custom-kamelet-integration"), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))

		Eventually(IntegrationLogs(ns, "timer-custom-kamelet-integration")).Should(ContainSubstring("hello world"))

		// Remove the custom repository
		Expect(Kamel("kamelet", "remove-repo",
			"github:apache/camel-k/e2e/global/common/files/kamelets",
			"-n", ns,
			"-x", operatorID).Execute()).To(Succeed())
	})
}

func removeKamelet(name string, ns string) {
	kamelet := Kamelet(name, ns)()
	TestClient().Delete(TestContext, kamelet)
}
