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

package advanced

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKameletFromCustomRepository(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		Expect(KamelInstallWithID(t, operatorID, ns).Execute()).To(Succeed())
		Eventually(PlatformPhase(t, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		kameletName := "timer-custom-source"
		removeKamelet(t, kameletName, ns)

		Eventually(Kamelet(t, kameletName, ns)).Should(BeNil())
		// Add the custom repository
		Expect(Kamel(t, "kamelet", "add-repo",
			"github:squakez/ck-kamelet-test-repo/kamelets",
			"-n", ns,
			"-x", operatorID).Execute()).To(Succeed())

		Expect(KamelRunWithID(t, operatorID, ns, "files/TimerCustomKameletIntegration.java").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(t, ns, "timer-custom-kamelet-integration"), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(t, ns, "timer-custom-kamelet-integration")).Should(ContainSubstring("hello world"))

		// Remove the custom repository
		Expect(Kamel(t, "kamelet", "remove-repo",
			"github:squakez/ck-kamelet-test-repo/kamelets",
			"-n", ns,
			"-x", operatorID).Execute()).To(Succeed())
	})
}

func removeKamelet(t *testing.T, name string, ns string) {
	kamelet := Kamelet(t, name, ns)()
	TestClient(t).Delete(TestContext, kamelet)
}
