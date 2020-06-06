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
	"testing"

	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunSimpleExamples(t *testing.T) {

	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("run java", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/Java.java").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run java from GitHub", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "github:apache/camel-k/e2e/common/files/Java.java").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run java from GitHub (RAW)", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "https://raw.githubusercontent.com/apache/camel-k/master/e2e/common/files/Java.java").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run java with properties", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/Prop.java", "--property-file", "files/prop.properties").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "prop"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "prop", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "prop"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run xml", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/xml.xml").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "xml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "xml", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "xml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run groovy", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/groovy.groovy").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "groovy"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "groovy", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "groovy"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run js", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/js.js").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "js"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "js", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "js"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run kotlin", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/kotlin.kts").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "kotlin"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "kotlin", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "kotlin"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run yaml", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "yaml", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run yaml Quarkus", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "--name", "yaml-quarkus", "files/yaml.yaml", "-t", "quarkus.enabled=true").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "yaml-quarkus"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "yaml-quarkus", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "yaml-quarkus"), TestTimeoutShort).Should(ContainSubstring("powered by Quarkus"))
			Eventually(IntegrationLogs(ns, "yaml-quarkus"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run polyglot", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "--name", "polyglot", "files/js-polyglot.js", "files/yaml-polyglot.yaml").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "polyglot"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "polyglot", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "polyglot"), TestTimeoutShort).Should(ContainSubstring("Magicpolyglot-yaml"))
			Eventually(IntegrationLogs(ns, "polyglot"), TestTimeoutShort).Should(ContainSubstring("Magicpolyglot-js"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

	})
}
