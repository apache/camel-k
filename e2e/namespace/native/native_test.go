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

package native

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

var (
	withFastJarLayout = KitWithLabels(map[string]string{v1.IntegrationKitLayoutLabel: v1.IntegrationKitLayoutFastJar})
	withNativeLayout  = KitWithLabels(map[string]string{v1.IntegrationKitLayoutLabel: v1.IntegrationKitLayoutNative})
)

func TestNativeIntegrations(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-quarkus-native"
		Expect(KamelInstallWithID(operatorID, ns,
			"--build-timeout", "15m0s",
			"--operator-resources", "limits.memory=4Gi",
		).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("unsupported integration source language", func(t *testing.T) {
			name := "unsupported-java"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
				"-t", "quarkus.package-type=native",
			).Execute()).To(Succeed())

			Eventually(IntegrationPhase(ns, name)).Should(Equal(v1.IntegrationPhaseError))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionKitAvailable)).Should(Equal(corev1.ConditionFalse))
		})

		t.Run("automatic rollout deployment from fast-jar to native kit", func(t *testing.T) {
			name := "jvm-to-native"
			Expect(KamelRunWithID(operatorID, ns, "files/yaml.yaml", "--name", name,
				"-t", "quarkus.package-type=fast-jar",
				"-t", "quarkus.package-type=native",
			).Execute()).To(Succeed())

			// Check that two Kits are created with distinct layout
			Eventually(Kits(ns, withFastJarLayout)).Should(HaveLen(1))
			Eventually(Kits(ns, withNativeLayout)).Should(HaveLen(1))

			// Check the fast-jar Kit is ready
			Eventually(Kits(ns, withFastJarLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutVeryLong).Should(HaveLen(1))

			fastJarKit := Kits(ns, withFastJarLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			// Check the Integration uses the fast-jar Kit
			Eventually(IntegrationKit(ns, name), TestTimeoutShort).Should(Equal(fastJarKit.Name))
			// Check the Integration Pod uses the fast-jar Kit
			Eventually(IntegrationPodImage(ns, name)).Should(Equal(fastJarKit.Status.Image))

			// Check the Integration is ready
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Check the native Kit is ready
			Eventually(Kits(ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutLong).Should(HaveLen(1))

			nativeKit := Kits(ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			// Check the Integration uses the native Kit
			Eventually(IntegrationKit(ns, name), TestTimeoutShort).Should(Equal(nativeKit.Name))
			// Check the Integration Pod uses the native Kit
			Eventually(IntegrationPodImage(ns, name)).Should(Equal(nativeKit.Status.Image))

			// Check the Integration is still ready
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
