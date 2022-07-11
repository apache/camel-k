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

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestRunExamplesFromGitHub(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("run java from GitHub", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "github:apache/camel-k/release-1.9.x/e2e/common/files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run java from GitHub (RAW)", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "https://raw.githubusercontent.com/apache/camel-k/release-1.9.x/e2e/common/files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run from GitHub Gist (ID)", func(t *testing.T) {
			name := "github-gist-id"
			Expect(Kamel("run", "-n", ns, "--name", name, "gist:e2c3f9a5fd0d9e79b21b04809786f17a").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Tick!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run from GitHub Gist (URL)", func(t *testing.T) {
			name := "github-gist-url"
			Expect(Kamel("run", "-n", ns, "--name", name, "https://gist.github.com/lburgazzoli/e2c3f9a5fd0d9e79b21b04809786f17a").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Tick!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunAndUpdate(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		name := "run"
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magic default"))

		// Re-run the Integration with an updated configuration
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name, "-p", "property=value").Execute()).To(Succeed())

		// Check the Deployment has progressed successfully
		Eventually(DeploymentCondition(ns, name, appsv1.DeploymentProgressing), TestTimeoutShort).Should(MatchFields(IgnoreExtras, Fields{
			"Status": Equal(corev1.ConditionTrue),
			"Reason": Equal("NewReplicaSetAvailable"),
		}))

		// Check the new configuration is taken into account
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magic value"))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
