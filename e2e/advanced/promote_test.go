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

package advanced

import (
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKamelCLIPromote(t *testing.T) {
	t.Parallel()

	one := int64(1)
	two := int64(2)
	// Dev environment namespace
	WithNewTestNamespace(t, func(g *WithT, nsDev string) {
		operatorDevID := "camel-k-cli-promote-dev"
		g.Expect(CopyCamelCatalog(t, nsDev, operatorDevID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, nsDev, operatorDevID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorDevID, nsDev).Execute()).To(Succeed())
		g.Eventually(SelectedPlatformPhase(t, nsDev, operatorDevID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		// Dev content configmap
		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "I am development configmap!"
		CreatePlainTextConfigmap(t, nsDev, "my-cm-promote", cmData)
		// Dev secret
		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret development"
		CreatePlainTextSecret(t, nsDev, "my-sec-promote", secData)

		t.Run("plain integration dev", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, operatorDevID, nsDev, "./files/promote-route.groovy",
				"--config", "configmap:my-cm-promote",
				"--config", "secret:my-sec-promote",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, nsDev, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationObservedGeneration(t, nsDev, "promote-route")).Should(Equal(&one))
			//g.Eventually(IntegrationConditionStatus(t, nsDev, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, nsDev, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am development configmap!"))
			g.Eventually(IntegrationLogs(t, nsDev, "promote-route"), TestTimeoutShort).Should(ContainSubstring("very top secret development"))
		})

		t.Run("kamelet integration dev", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, operatorDevID, nsDev, "my-own-timer-source")()).To(Succeed())
			g.Expect(KamelRunWithID(t, operatorDevID, nsDev, "./files/timer-kamelet-usage.groovy").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, nsDev, "timer-kamelet-usage"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, nsDev, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
		})

		t.Run("binding dev", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, operatorDevID, nsDev, "kb-timer-source")()).To(Succeed())
			g.Expect(KamelBindWithID(t, operatorDevID, nsDev, "kb-timer-source", "log:info", "-p", "source.message=my-kamelet-binding-rocks").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, nsDev, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, nsDev, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks"))
		})

		// Prod environment namespace
		WithNewTestNamespace(t, func(g *WithT, nsProd string) {
			operatorProdID := "camel-k-cli-promote-prod"
			g.Expect(CopyCamelCatalog(t, nsProd, operatorProdID)).To(Succeed())
			g.Expect(CopyIntegrationKits(t, nsProd, operatorProdID)).To(Succeed())
			g.Expect(KamelInstallWithID(t, operatorProdID, nsProd).Execute()).To(Succeed())
			g.Eventually(PlatformPhase(t, nsProd), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

			t.Run("no configmap in destination", func(t *testing.T) {
				g.Expect(Kamel(t, "promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			// Prod content configmap
			var cmData = make(map[string]string)
			cmData["my-configmap-key"] = "I am production!"
			CreatePlainTextConfigmap(t, nsProd, "my-cm-promote", cmData)

			t.Run("no secret in destination", func(t *testing.T) {
				g.Expect(Kamel(t, "promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			// Prod secret
			var secData = make(map[string]string)
			secData["my-secret-key"] = "very top secret production"
			CreatePlainTextSecret(t, nsProd, "my-sec-promote", secData)

			t.Run("plain integration promotion", func(t *testing.T) {
				g.Expect(Kamel(t, "promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).To(Succeed())
				g.Eventually(IntegrationObservedGeneration(t, nsProd, "promote-route")).Should(Equal(&one))
				g.Eventually(IntegrationPodPhase(t, nsProd, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, nsProd, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, nsProd, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am production!"))
				g.Eventually(IntegrationLogs(t, nsProd, "promote-route"), TestTimeoutShort).Should(ContainSubstring("very top secret production"))
				// They must use the same image
				g.Expect(IntegrationPodImage(t, nsProd, "promote-route")()).Should(Equal(IntegrationPodImage(t, nsDev, "promote-route")()))
			})

			t.Run("plain integration promotion update", func(t *testing.T) {
				// We need to update the Integration CR in order the operator to restart it both in dev and prod envs
				g.Expect(KamelRunWithID(t, operatorDevID, nsDev, "./files/promote-route-edited.groovy", "--name", "promote-route",
					"--config", "configmap:my-cm-promote").Execute()).To(Succeed())
				// The generation has to be incremented
				g.Eventually(IntegrationObservedGeneration(t, nsDev, "promote-route")).Should(Equal(&two))
				g.Eventually(IntegrationPodPhase(t, nsDev, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, nsDev, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, nsDev, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am development configmap!"))
				// Update the configmap only in prod
				var cmData = make(map[string]string)
				cmData["my-configmap-key"] = "I am production, but I was updated!"
				UpdatePlainTextConfigmap(t, nsProd, "my-cm-promote", cmData)
				// Promote the edited Integration
				g.Expect(Kamel(t, "promote", "-n", nsDev, "promote-route", "--to", nsProd).Execute()).To(Succeed())
				// The generation has to be incremented also in prod
				g.Eventually(IntegrationObservedGeneration(t, nsDev, "promote-route")).Should(Equal(&two))
				g.Eventually(IntegrationPodPhase(t, nsProd, "promote-route"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, nsProd, "promote-route", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, nsProd, "promote-route"), TestTimeoutShort).Should(ContainSubstring("I am production, but I was updated!"))
				// They must use the same image
				g.Expect(IntegrationPodImage(t, nsProd, "promote-route")()).Should(Equal(IntegrationPodImage(t, nsDev, "promote-route")()))
			})

			t.Run("no kamelet in destination", func(t *testing.T) {
				g.Expect(Kamel(t, "promote", "-n", nsDev, "timer-kamelet-usage", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			t.Run("kamelet integration promotion", func(t *testing.T) {
				g.Expect(CreateTimerKamelet(t, operatorProdID, nsProd, "my-own-timer-source")()).To(Succeed())
				g.Expect(Kamel(t, "promote", "-n", nsDev, "timer-kamelet-usage", "--to", nsProd).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, nsProd, "timer-kamelet-usage"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, nsProd, "timer-kamelet-usage"), TestTimeoutShort).Should(ContainSubstring("Hello world"))
				// They must use the same image
				g.Expect(IntegrationPodImage(t, nsProd, "timer-kamelet-usage")()).Should(Equal(IntegrationPodImage(t, nsDev, "timer-kamelet-usage")()))
			})

			t.Run("no kamelet for binding in destination", func(t *testing.T) {
				g.Expect(Kamel(t, "promote", "-n", nsDev, "kb-timer-source-to-log", "--to", nsProd).Execute()).NotTo(Succeed())
			})

			t.Run("binding promotion", func(t *testing.T) {
				g.Expect(CreateTimerKamelet(t, operatorProdID, nsProd, "kb-timer-source")()).To(Succeed())
				g.Expect(Kamel(t, "promote", "-n", nsDev, "kb-timer-source-to-log", "--to", nsProd).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, nsProd, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, nsProd, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks"))
				// They must use the same image
				g.Expect(IntegrationPodImage(t, nsProd, "kb-timer-source-to-log")()).Should(Equal(IntegrationPodImage(t, nsDev, "kb-timer-source-to-log")()))

				//Binding update
				g.Expect(KamelBindWithID(t, operatorDevID, nsDev, "kb-timer-source", "log:info", "-p", "source.message=my-kamelet-binding-rocks-again").Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, nsDev, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, nsDev, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks-again"))
				g.Expect(Kamel(t, "promote", "-n", nsDev, "kb-timer-source-to-log", "--to", nsProd).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, nsProd, "kb-timer-source-to-log"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, nsProd, "kb-timer-source-to-log"), TestTimeoutShort).Should(ContainSubstring("my-kamelet-binding-rocks-again"))
			})
		})
	})
}
