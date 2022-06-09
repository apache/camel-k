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
	"context"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/e2e/support/util"
)

func TestRunDevMode(t *testing.T) {

	/*
	 * TODO
	 * The changing of the yaml file constant from "string" to "magic" is not being
	 * picked up when deploying on OCP4 and so the test is failing.
	 *
	 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
	 */
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-dev-mode"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("run yaml dev mode", func(t *testing.T) {
			RegisterTestingT(t)
			ctx, cancel := context.WithCancel(TestContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "files/yaml.yaml")

			kamelRun := KamelRunWithContext(ctx, operatorID, ns, file, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "yaml" in phase Running`, "Magicstring!", "Magicjordan!")

			args := os.Args
			defer func() { os.Args = args }()

			globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
			if globalTest {
				os.Args = []string{"kamel", "run", "-n", ns, file, "--dev"}
			} else {
				os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, file, "--dev"}
			}
			go kamelRun.Execute()

			Eventually(logScanner.IsFound(`integration "yaml" in phase Running`), TestTimeoutMedium).Should(BeTrue())
			Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
			Expect(logScanner.IsFound("Magicjordan!")()).To(BeFalse())

			util.ReplaceInFile(t, file, "string!", "jordan!")
			Eventually(logScanner.IsFound("Magicjordan!"), TestTimeoutMedium).Should(BeTrue())
		})

		t.Run("run yaml remote dev mode", func(t *testing.T) {
			RegisterTestingT(t)
			ctx, cancel := context.WithCancel(TestContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			remoteFile := "https://raw.githubusercontent.com/apache/camel-k/b29333f0a878d5d09fb3965be8fe586d77dd95d0/e2e/common/files/yaml.yaml"
			kamelRun := KamelRunWithContext(ctx, operatorID, ns, remoteFile, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, "Magicstring!")

			args := os.Args
			defer func() { os.Args = args }()

			globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
			if globalTest {
				os.Args = []string{"kamel", "run", "-n", ns, remoteFile, "--dev"}
			} else {
				os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, remoteFile, "--dev"}
			}

			go kamelRun.Execute()

			Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
		})

		t.Run("Dev mode resource file generated configmap", func(t *testing.T) {
			var tmpFile *os.File
			var err error
			if tmpFile, err = ioutil.TempFile("", "camel-k-"); err != nil {
				t.Error(err)
			}
			assert.Nil(t, tmpFile.Close())
			assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte("Hello from test!"), 0o644))

			RegisterTestingT(t)
			ctx, cancel := context.WithCancel(TestContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "../config/files/resource-file-location-route.groovy")

			kamelRun := KamelRunWithContext(ctx, operatorID, ns, file, "--dev", "--resource", fmt.Sprintf("file:%s@/tmp/file.txt", tmpFile.Name()))
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "resource-file-location-route" in phase Running`,
				"Hello from test!", "Goodbye from test!")

			args := os.Args
			defer func() { os.Args = args }()

			globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
			if globalTest {
				os.Args = []string{"kamel", "run", "-n", ns, file, "--dev", "--resource", fmt.Sprintf("file:%s@/tmp/file.txt", tmpFile.Name())}
			} else {
				os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, file, "--dev", "--resource", fmt.Sprintf("file:%s@/tmp/file.txt", tmpFile.Name())}
			}

			go kamelRun.Execute()

			Eventually(logScanner.IsFound(`integration "resource-file-location-route" in phase Running`), TestTimeoutMedium).Should(BeTrue())
			Eventually(logScanner.IsFound("Hello from test!"), TestTimeoutMedium).Should(BeTrue())
			Expect(logScanner.IsFound("Goodbye from test!")()).To(BeFalse())

			// cool, now let's change the file to confirm the sync take place
			assert.Nil(t, ioutil.WriteFile(tmpFile.Name(), []byte("Goodbye from test!"), 0o644))
			Eventually(logScanner.IsFound("Goodbye from test!"), TestTimeoutMedium).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			// When the integration is deleted, then, also the autogenerated configmaps must be cleaned
			Eventually(AutogeneratedConfigmapsCount(ns), TestTimeoutShort).Should(Equal(0))
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
			RegisterTestingT(t)

			// First run (warm up)
			Expect(KamelRunWithID(operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(ns, "yaml")).Should(BeNil())

			// Second run (rebuild)
			ctx, cancel := context.WithCancel(TestContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "files/yaml.yaml")

			kamelRun := KamelRunWithContext(ctx, operatorID, ns, file, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "yaml" in phase Running`, "Magicstring!")

			args := os.Args
			defer func() { os.Args = args }()

			globalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
			if globalTest {
				os.Args = []string{"kamel", "run", "-n", ns, file, "--dev"}
			} else {
				os.Args = []string{"kamel", "run", "-n", ns, "--operator-id", operatorID, file, "--dev"}
			}

			go kamelRun.Execute()

			// Second run should start up within a few seconds
			timeout := 10 * time.Second
			Eventually(logScanner.IsFound(`integration "yaml" in phase Running`), timeout).Should(BeTrue())
			Eventually(logScanner.IsFound("Magicstring!"), timeout).Should(BeTrue())
		})
	})
}
