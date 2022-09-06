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
	"strings"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelCLILog(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-cli-log"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("check integration log", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			// first line of the integration logs
			firstLine := strings.Split(IntegrationLogs(ns, "yaml")(), "\n")[0]
			podName := IntegrationPod(ns, "yaml")().Name

			logsCLI := GetOutputStringAsync(Kamel("log", "yaml", "-n", ns))
			Eventually(logsCLI).Should(ContainSubstring("Monitoring pod " + podName))
			Eventually(logsCLI).Should(ContainSubstring(firstLine))

			logs := strings.Split(IntegrationLogs(ns, "yaml")(), "\n")
			lastLine := logs[len(logs)-1]

			logsCLI = GetOutputStringAsync(Kamel("log", "yaml", "-n", ns, "--tail", "5"))
			Eventually(logsCLI).Should(ContainSubstring("Monitoring pod " + podName))
			Eventually(logsCLI).Should(ContainSubstring(lastLine))
		})
	})
}
