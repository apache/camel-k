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

package cli

import (
	"context"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKamelCLILog(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("check integration log", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml", "--name", "log-yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "log-yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			// first line of the integration logs
			firstLine := strings.Split(IntegrationLogs(t, ctx, ns, "log-yaml")(), "\n")[0]
			podName := IntegrationPod(t, ctx, ns, "log-yaml")().Name

			logsCLI := GetOutputStringAsync(Kamel(t, ctx, "log", "log-yaml", "-n", ns))
			g.Eventually(logsCLI).Should(ContainSubstring("Monitoring pod " + podName))
			g.Eventually(logsCLI).Should(ContainSubstring(firstLine))

			logs := strings.Split(IntegrationLogs(t, ctx, ns, "log-yaml")(), "\n")
			lastLine := logs[len(logs)-1]

			logsCLI = GetOutputStringAsync(Kamel(t, ctx, "log", "log-yaml", "-n", ns, "--tail", "5"))
			g.Eventually(logsCLI).Should(ContainSubstring("Monitoring pod " + podName))
			g.Eventually(logsCLI).Should(ContainSubstring(lastLine))
		})
	})
}
