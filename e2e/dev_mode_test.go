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

package e2e

import (
	"context"
	"io"
	"testing"

	"github.com/apache/camel-k/e2e/util"
	. "github.com/onsi/gomega"
)

func TestRunDevMode(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("run yaml dev mode", func(t *testing.T) {
			RegisterTestingT(t)
			ctx, cancel := context.WithCancel(testContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			file := util.MakeTempCopy(t, "files/yaml.yaml")

			kamelRun := kamelWithContext(ctx, "run", "-n", ns, file, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, `integration "yaml" in phase Running`, "Magicstring!", "Magicjordan!")

			go kamelRun.Execute()

			Eventually(logScanner.IsFound(`integration "yaml" in phase Running`), testTimeoutMedium).Should(BeTrue())
			Eventually(logScanner.IsFound("Magicstring!"), testTimeoutMedium).Should(BeTrue())
			Expect(logScanner.IsFound("Magicjordan!")()).To(BeFalse())

			util.ReplaceInFile(t, file, "string!", "jordan!")
			Eventually(logScanner.IsFound("Magicjordan!"), testTimeoutMedium).Should(BeTrue())
		})

		t.Run("run yaml remote dev mode", func(t *testing.T) {
			RegisterTestingT(t)
			ctx, cancel := context.WithCancel(testContext)
			defer cancel()
			piper, pipew := io.Pipe()
			defer pipew.Close()
			defer piper.Close()

			remoteFile := "https://github.com/apache/camel-k/raw/e80eb5353cbccf47c89a9f0a1c68ffbe3d0f1521/e2e/files/yaml.yaml"
			kamelRun := kamelWithContext(ctx, "run", "-n", ns, remoteFile, "--dev")
			kamelRun.SetOut(pipew)

			logScanner := util.NewLogScanner(ctx, piper, "Magicstring!")

			go kamelRun.Execute()

			Eventually(logScanner.IsFound("Magicstring!"), testTimeoutMedium).Should(BeTrue())
		})
	})
}
