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

package misc

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKameletClasspathLoading(t *testing.T) {
	RegisterTestingT(t)

	// Basic
	t.Run("test basic case", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/TimerKameletIntegration.java", "-t", "kamelets.enabled=false",
			"--resource", "file:files/my-timer-source.kamelet.yaml@/kamelets/my-timer-source.kamelet.yaml",
			"-p camel.component.kamelet.location=file:/kamelets",
			"-d", "camel:yaml-dsl",
			// kamelet dependencies
			"-d", "camel:timer").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-kamelet-integration"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, "timer-kamelet-integration")).Should(ContainSubstring("important message"))
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
