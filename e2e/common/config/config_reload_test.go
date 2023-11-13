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

package config

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestConfigmapHotReload(t *testing.T) {
	RegisterTestingT(t)

	name := RandomizedSuffixName("config-configmap-route")

	var cmData = make(map[string]string)
	cmData["my-configmap-key"] = "my configmap content"
	CreatePlainTextConfigmap(ns, "my-hot-cm", cmData)

	Expect(KamelRunWithID(operatorID, ns,
		"./files/config-configmap-route.groovy",
		"--config",
		"configmap:my-hot-cm",
		"-t",
		"mount.hot-reload=true",
		"--name",
		name,
	).Execute()).To(Succeed())
	Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("my configmap content"))

	cmData["my-configmap-key"] = "my configmap content updated"
	UpdatePlainTextConfigmap(ns, "my-hot-cm", cmData)
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("my configmap content updated"))

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestConfigmapHotReloadDefault(t *testing.T) {
	RegisterTestingT(t)

	name := RandomizedSuffixName("config-configmap-route")

	var cmData = make(map[string]string)
	cmData["my-configmap-key"] = "my configmap content"
	CreatePlainTextConfigmap(ns, "my-hot-cm-2", cmData)

	Expect(KamelRunWithID(operatorID, ns, "./files/config-configmap-route.groovy",
		"--config",
		"configmap:my-hot-cm-2",
		"--name",
		name,
	).Execute()).To(Succeed())
	Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("my configmap content"))

	cmData["my-configmap-key"] = "my configmap content updated"
	UpdatePlainTextConfigmap(ns, "my-hot-cm-2", cmData)
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(Not(ContainSubstring("my configmap content updated")))

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func TestSecretHotReload(t *testing.T) {
	RegisterTestingT(t)

	name := RandomizedSuffixName("config-secret-route")

	var secData = make(map[string]string)
	secData["my-secret-key"] = "very top secret"
	CreatePlainTextSecret(ns, "my-hot-sec", secData)

	Expect(KamelRunWithID(operatorID, ns, "./files/config-secret-route.groovy",
		"--config",
		"secret:my-hot-sec",
		"-t",
		"mount.hot-reload=true",
		"--name",
		name,
	).Execute()).To(Succeed())
	Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("very top secret"))

	secData["my-secret-key"] = "very top secret updated"
	UpdatePlainTextSecret(ns, "my-hot-sec", secData)
	Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("very top secret updated"))

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
