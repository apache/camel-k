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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestKamelCLIPromote(t *testing.T) {
	one := int64(1)
	two := int64(2)
	// Dev environment namespace
	WithNewTestNamespace(t, func(nsDev string) {
		operatorDevID := "camel-k-cli-promote-dev"
		Expect(KamelInstallWithID(operatorDevID, nsDev).Execute()).To(Succeed())

		// Dev content configmap
		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "I am development configmap!"
		CreatePlainTextConfigmap(nsDev, "my-cm", cmData)
		// Dev secret
		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret development"
		CreatePlainTextSecret(nsDev, "my-sec", secData)

		t.Run("plain integration dev", func(t *testing.T) {
			Expect(KamelRunWithID(operatorDevID, nsDev, "./files/promote-route.groovy",
				"--config", "configmap:my-cm",
				"--config", "secret:my-sec",
			).Execute()).To(Succeed())
			Eventually(IntegrationObservedGeneration(nsDev, "promote-route")).Should(Equal(&one))
			Eventually(IntegrationPodPhase(nsDev, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(nsDev, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(nsDev, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am development configmap!"))
			Eventually(IntegrationLogs(nsDev, "promote-route"), TestTimeoutShort).Should(ContainSubstring("very top secret development"))
		})

		t.Run("kamelet integration dev", func(t *testing.T) {
			Expect(CreateTimerKamelet(nsDev, "my-own-timer-source")()).To(Succeed())
			Expect(KamelRunWithID(operatorDevID, nsDev, "./files/timer-kamelet-usage.groovy").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(nsDev, "timer-kamelet-usage"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(nsDev, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
		})

		t.Run("kamelet binding dev", func(t *testing.T) {
			Expect(CreateTimerKamelet(nsDev, "kb-timer-source")()).To(Succeed())
			Expect(KamelBindWithID(operatorDevID, nsDev, "kb-timer-source", "log:info", "-p", "source.message=my-kamelet-binding-rocks").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(nsDev, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(nsDev, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks"))
		})

		// Prod environment namespace
		WithNewTestNamespace(t, func(nsProd string) {
			operatorProdID := "camel-k-cli-promote-prod"
			Expect(KamelInstallWithID(operatorProdID, nsProd).Execute()).To(Succeed())
			Eventually(PlatformPhase(nsProd), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

			t.Run("no configmap in destination", func(t *testing.T) {
				Expect(Kamel("promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			// Prod content configmap
			var cmData = make(map[string]string)
			cmData["my-configmap-key"] = "I am production!"
			CreatePlainTextConfigmap(nsProd, "my-cm", cmData)

			t.Run("no secret in destination", func(t *testing.T) {
				Expect(Kamel("promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			// Prod secret
			var secData = make(map[string]string)
			secData["my-secret-key"] = "very top secret production"
			CreatePlainTextSecret(nsProd, "my-sec", secData)

			t.Run("plain integration promotion", func(t *testing.T) {
				Expect(Kamel("promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).To(Succeed())
				Eventually(IntegrationObservedGeneration(nsProd, "promote-route")).Should(Equal(&one))
				Eventually(IntegrationPodPhase(nsProd, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationConditionStatus(nsProd, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				Eventually(IntegrationLogs(nsProd, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am production!"))
				Eventually(IntegrationLogs(nsProd, "promote-route"), TestTimeoutShort).Should(ContainSubstring("very top secret production"))
				// They must use the same image
				Expect(IntegrationPodImage(nsProd, "promote-route")()).Should(Equal(IntegrationPodImage(nsDev, "promote-route")()))
			})

			t.Run("plain integration promotion update", func(t *testing.T) {
				// We need to update the Integration CR in order the operator to restart it both in dev and prod envs
				Expect(KamelRunWithID(operatorDevID, nsDev, "./files/promote-route-edited.groovy", "--name", "promote-route",
					"--config", "configmap:my-cm").Execute()).To(Succeed())
				// The generation has to be incremented
				Eventually(IntegrationObservedGeneration(nsDev, "promote-route")).Should(Equal(&two))
				Eventually(IntegrationPodPhase(nsDev, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationConditionStatus(nsDev, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				Eventually(IntegrationLogs(nsDev, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am development configmap!"))
				// Update the configmap only in prod
				var cmData = make(map[string]string)
				cmData["my-configmap-key"] = "I am production, but I was updated!"
				UpdatePlainTextConfigmap(nsProd, "my-cm", cmData)
				// Promote the edited Integration
				Expect(Kamel("promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).To(Succeed())
				// The generation has to be incremented also in prod
				Eventually(IntegrationObservedGeneration(nsDev, "promote-route")).Should(Equal(&two))
				Eventually(IntegrationPodPhase(nsProd, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationConditionStatus(nsProd, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				Eventually(IntegrationLogs(nsProd, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am production, but I was updated!"))
				// They must use the same image
				Expect(IntegrationPodImage(nsProd, "promote-route")()).Should(Equal(IntegrationPodImage(nsDev, "promote-route")()))
			})

			t.Run("no kamelet in destination", func(t *testing.T) {
				Expect(Kamel("promote", "-n", nsDev, "timer-kamelet-usage", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			t.Run("kamelet integration promotion", func(t *testing.T) {
				Expect(CreateTimerKamelet(nsProd, "my-own-timer-source")()).To(Succeed())
				Expect(Kamel("promote", "-n", nsDev, "timer-kamelet-usage", "--to", nsProd).Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(nsProd, "timer-kamelet-usage"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(nsProd, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
				// They must use the same image
				Expect(IntegrationPodImage(nsProd, "timer-kamelet-usage")()).Should(Equal(IntegrationPodImage(nsDev, "timer-kamelet-usage")()))
			})

			t.Run("no kamelet for kameletbinding in destination", func(t *testing.T) {
				Expect(Kamel("promote", "-n", nsDev, "kb-timer-source-to-log", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			t.Run("kamelet binding promotion", func(t *testing.T) {
				Expect(CreateTimerKamelet(nsProd, "kb-timer-source")()).To(Succeed())
				Expect(Kamel("promote", "-n", nsDev, "kb-timer-source-to-log", "--to", nsProd).Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(nsProd, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(nsProd, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks"))
				// They must use the same image
				Expect(IntegrationPodImage(nsProd, "kb-timer-source-to-log")()).Should(Equal(IntegrationPodImage(nsDev, "kb-timer-source-to-log")()))

				// Kamelet Binding update
				Expect(KamelBindWithID(operatorDevID, nsDev, "kb-timer-source", "log:info", "-p", "source.message=my-kamelet-binding-rocks-again").Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(nsDev, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(nsDev, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks-again"))
				Expect(Kamel("promote", "-n", nsDev, "kb-timer-source-to-log", "--to", nsProd).Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(nsProd, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(nsProd, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks-again"))
			})
		})
	})
}
