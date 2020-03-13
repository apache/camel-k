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
	"fmt"
	"io"
	"path"
	"testing"

	"github.com/apache/camel-k/e2e/util"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunSimpleExamples(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())

		for _, lang := range camelv1.Languages {
			t.Run("init run "+string(lang), func(t *testing.T) {
				RegisterTestingT(t)
				dir := util.MakeTempDir(t)
				itName := fmt.Sprintf("init%s", string(lang))          // e.g. initjava
				fileName := fmt.Sprintf("%s.%s", itName, string(lang)) // e.g. initjava.java
				file := path.Join(dir, fileName)
				Expect(kamel("init", file).Execute()).Should(BeNil())
				Expect(kamel("run", "-n", ns, file).Execute()).Should(BeNil())
				Eventually(integrationPodPhase(ns, itName), testTimeoutMedium).Should(Equal(v1.PodRunning))
				Eventually(integrationLogs(ns, itName), testTimeoutShort).Should(ContainSubstring(languageInitExpectedString(lang)))
				Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
			})
		}

		t.Run("run java", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/Java.java").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "java"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "java"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run java with properties", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/Prop.java", "--property-file", "files/prop.properties").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "prop"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "prop"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run xml", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/xml.xml").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "xml"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "xml"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run groovy", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/groovy.groovy").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "groovy"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "groovy"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run js", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/js.js").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "js"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "js"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run kotlin", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/kotlin.kts").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "kotlin"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "kotlin"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run yaml", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/yaml.yaml").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "yaml"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "yaml"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run yaml Quarkus", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "--name", "yaml-quarkus", "files/yaml.yaml", "-t", "quarkus.enabled=true").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "yaml-quarkus"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "yaml-quarkus"), testTimeoutShort).Should(ContainSubstring("running on Quarkus"))
			Eventually(integrationLogs(ns, "yaml-quarkus"), testTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run polyglot", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "--name", "polyglot", "files/js-polyglot.js", "files/yaml-polyglot.yaml").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "polyglot"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "polyglot"), testTimeoutShort).Should(ContainSubstring("Magicpolyglot-yaml"))
			Eventually(integrationLogs(ns, "polyglot"), testTimeoutShort).Should(ContainSubstring("Magicpolyglot-js"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

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
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
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
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

	})
}


func languageInitExpectedString(lang camelv1.Language) string {
	langDesc := string(lang)
	if lang == camelv1.LanguageKotlin {
		langDesc = "kotlin"
	}
	return fmt.Sprintf(" Hello Camel K from %s", langDesc)
}
