// +build integration

// To enable compilation of this file in Goland, go to "File -> Settings -> Go -> Build Tags & Vendoring -> Build Tags -> Custom tags" and add "integration"

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
	"io"
	"os"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/e2e/support/util"
	. "github.com/onsi/gomega"
)

func TestRunDevMode(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("run yaml dev mode", func(t *testing.T) {
			RegisterTestingT(t)
			ctx, cancel := context.WithCancel(TestContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "files/yaml.yaml")

			kamelRun := KamelWithContext(ctx, "run", "-n", ns, file, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "yaml" in phase Running`, "Magicstring!", "Magicjordan!")

			args := os.Args
			defer func() { os.Args = args }()
			os.Args = []string{"kamel", "run", "-n", ns, file, "--dev"}
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
			kamelRun := KamelWithContext(ctx, "run", "-n", ns, remoteFile, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, "Magicstring!")

			args := os.Args
			defer func() { os.Args = args }()
			os.Args = []string{"kamel", "-n", ns, "run", remoteFile, "--dev"}
			go kamelRun.Execute()

			Eventually(logScanner.IsFound("Magicstring!"), TestTimeoutMedium).Should(BeTrue())
		})
	})
}
