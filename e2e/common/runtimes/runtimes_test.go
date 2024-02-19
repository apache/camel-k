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

package runtimes

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestSourceLessIntegrations(t *testing.T) {
	RegisterTestingT(t)
	var cmData = make(map[string]string)
	cmData["my-file.txt"] = "Hello World!"
	CreatePlainTextConfigmap(ns, "my-cm-sourceless", cmData)

	t.Run("Camel Main", func(t *testing.T) {
		itName := "my-camel-main-v1"
		Expect(KamelRunWithID(operatorID, ns, "--image", "docker.io/squakez/my-camel-main:1.0.0", "--resource", "configmap:my-cm-sourceless@/tmp/app/data").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, itName), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, itName, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationCondition(ns, itName, v1.IntegrationConditionTraitInfo)().Message).Should(Equal("explicitly disabled by the platform: container image was not built via Camel K operator"))
		Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring(cmData["my-file.txt"]))
		Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring("Apache Camel (Main)"))
	})

	t.Run("Camel Spring Boot", func(t *testing.T) {
		itName := "my-camel-sb-v1"
		Expect(KamelRunWithID(operatorID, ns, "--image", "docker.io/squakez/my-camel-sb:1.0.0", "--resource", "configmap:my-cm-sourceless@/tmp/app/data").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, itName), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, itName, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationCondition(ns, itName, v1.IntegrationConditionTraitInfo)().Message).Should(Equal("explicitly disabled by the platform: container image was not built via Camel K operator"))
		Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring(cmData["my-file.txt"]))
		Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring("Spring Boot"))
	})

	t.Run("Camel Quarkus", func(t *testing.T) {
		itName := "my-camel-quarkus-v1"
		Expect(KamelRunWithID(operatorID, ns, "--image", "docker.io/squakez/my-camel-quarkus:1.0.0", "--resource", "configmap:my-cm-sourceless@/tmp/app/data").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, itName), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, itName, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationCondition(ns, itName, v1.IntegrationConditionTraitInfo)().Message).Should(Equal("explicitly disabled by the platform: container image was not built via Camel K operator"))
		Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring(cmData["my-file.txt"]))
		Eventually(IntegrationLogs(ns, itName), TestTimeoutShort).Should(ContainSubstring("powered by Quarkus"))
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
