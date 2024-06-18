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
	"io"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/e2e/support/util"
)

func TestRunDevMode(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("run yaml dev mode", func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "files/yaml.yaml")
			name := RandomizedSuffixName("yaml")

			kamelRun := KamelRunWithContext(t, ctx, operatorID, ns, file, "--name", name, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "`+name+`" in phase Running`, "Magicstring!", "Magicjordan!")

			args := os.Args
			defer func() { os.Args = args }()

			os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, file, "--name", name, "--dev"}
			go kamelRun.Execute()

			g.Eventually(logScanner.IsFound(`integration "`+name+`" in phase Running`), TestTimeoutMedium).Should(BeTrue())
			g.Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
			g.Expect(logScanner.IsFound("Magicjordan!")()).To(BeFalse())

			util.ReplaceInFile(t, file, "string!", "jordan!")
			g.Eventually(logScanner.IsFound("Magicjordan!"), TestTimeoutMedium).Should(BeTrue())
		})

		t.Run("run yaml remote dev mode", func(t *testing.T) {
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			remoteFile := "https://raw.githubusercontent.com/apache/camel-k/b29333f0a878d5d09fb3965be8fe586d77dd95d0/e2e/common/files/yaml.yaml"
			name := RandomizedSuffixName("yaml")
			kamelRun := KamelRunWithContext(t, ctx, operatorID, ns, remoteFile, "--name", name, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, "Magicstring!")

			args := os.Args
			defer func() { os.Args = args }()

			os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, remoteFile, "--name", name, "--dev"}

			go kamelRun.Execute()

			g.Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
		})

		// This test makes sure that `kamel run --dev` runs in seconds after initial build is
		// already done for the same integration.
		t.Run("dev mode rebuild in seconds", func(t *testing.T) {
			/*
			 * !!! NOTE !!!
			 * If you find this test flaky, instead of thinking it as simply unstable, investigate
			 * why it does not finish in a few seconds and remove the bottlenecks which are lagging
			 * the integration startup.
			 */
			name := RandomizedSuffixName("yaml")

			// First run (warm up)
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Expect(Kamel(t, ctx, "delete", name, "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, name)).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, name), TestTimeoutMedium).Should(BeNil())

			// Second run (rebuild)
			ctx, cancel := context.WithCancel(ctx)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "files/yaml.yaml")

			kamelRun := KamelRunWithContext(t, ctx, operatorID, ns, file, "--name", name, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "`+name+`" in phase Running`, "Magicstring!")

			args := os.Args
			defer func() { os.Args = args }()

			os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, file, "--name", name, "--dev"}

			go kamelRun.Execute()

			// Second run should start up within a few seconds
			timeout := 20 * time.Second
			g.Eventually(logScanner.IsFound(`integration "`+name+`" in phase Running`), timeout).Should(BeTrue())
			g.Eventually(logScanner.IsFound("Magicstring!"), timeout).Should(BeTrue())
		})

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
