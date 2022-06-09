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
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelCLIDelete(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-cli-delete"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("delete running integration", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel("delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(ns, "yaml")).Should(BeNil())
		})

		t.Run("delete building integration", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Expect(Kamel("delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(ns, "yaml")).Should(BeNil())
		})

		t.Run("delete integration from csv", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "github:apache/camel-k/e2e/common/files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel("delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(ns, "yaml")).Should(BeNil())
		})

		t.Run("delete several integrations", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Expect(KamelRunWithID(operatorID, ns, "../files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel("delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(ns, "yaml")).Should(BeNil())
			Expect(Kamel("delete", "java", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "java")).Should(BeNil())
			Eventually(IntegrationPod(ns, "java")).Should(BeNil())
		})

		t.Run("delete all integrations", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Expect(KamelRunWithID(operatorID, ns, "../files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(ns, "yaml")).Should(BeNil())
			Eventually(Integration(ns, "java")).Should(BeNil())
			Eventually(IntegrationPod(ns, "java")).Should(BeNil())
		})
	})
}
