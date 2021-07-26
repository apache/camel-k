// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "knative"

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
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKameletClasspathLoading(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		kameletName := "timer-source"
		removeKamelet(kameletName , ns)

		Eventually(Kamelet(kameletName, ns)).Should(BeNil())

		Expect(Kamel("run", "files/TimerKameletIntegration.java", "-n", ns, "-t", "kamelets.enabled=false",
			"--resource", "file:files/timer-source.kamelet.yaml@/kamelets/timer-source.kamelet.yaml",
			"-p camel.component.kamelet.location=file:/kamelets",
			"-d", "camel:yaml-dsl",
			// kamelet dependencies
			"-d", "camel:timer").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-kamelet-integration"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

		Eventually(IntegrationLogs(ns, "timer-kamelet-integration")).Should(ContainSubstring("important message"))

		// Cleanup
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}

func removeKamelet(name string, ns string) {
	kamelet := Kamelet(name, ns)()
	TestClient().Delete(TestContext, kamelet)
}
