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
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKamelCLIDelete(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		t.Run("delete running integration", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel(t, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(t, ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(t, ns, "yaml"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete building integration", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			Expect(Kamel(t, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(t, ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(t, ns, "yaml"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete several integrations", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodPhase(t, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel(t, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(t, ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(t, ns, "yaml"), TestTimeoutLong).Should(BeNil())
			Expect(Kamel(t, "delete", "java", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(t, ns, "java")).Should(BeNil())
			Eventually(IntegrationPod(t, ns, "java"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete all integrations", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodPhase(t, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			Eventually(Integration(t, ns, "yaml")).Should(BeNil())
			Eventually(IntegrationPod(t, ns, "yaml"), TestTimeoutLong).Should(BeNil())
			Eventually(Integration(t, ns, "java")).Should(BeNil())
			Eventually(IntegrationPod(t, ns, "java"), TestTimeoutLong).Should(BeNil())
		})

		Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
