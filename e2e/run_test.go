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
	"testing"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunSimpleExamples(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())

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

	})
}
