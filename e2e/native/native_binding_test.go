//go:build integration && !high_memory
// +build integration,!high_memory

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
	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"testing"
)

func TestNativeBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-native-binding"
		Expect(KamelInstallWithIDAndKameletCatalog(operatorID, ns,
			"--build-timeout", "90m0s",
			"--maven-cli-option", "-Dquarkus.native.native-image-xmx=6g",
		).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		message := "Magicstring!"
		t.Run("binding with native build", func(t *testing.T) {
			bindingName := "native-binding"
			Expect(KamelBindWithID(operatorID, ns,
				"timer-source",
				"log-sink",
				"-p", "source.message="+message,
				"--annotation", "trait.camel.apache.org/quarkus.mode=native",
				"--annotation", "trait.camel.apache.org/builder.tasks-limit-memory=quarkus-native:6.5Gi",
				"--name", bindingName,
			).Execute()).To(Succeed())

			// ====================================
			// !!! THE MOST TIME-CONSUMING PART !!!
			// ====================================
			Eventually(Kits(ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutVeryLong).Should(HaveLen(1))

			nativeKit := Kits(ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			Eventually(IntegrationKit(ns, bindingName), TestTimeoutShort).Should(Equal(nativeKit.Name))

			Eventually(IntegrationPodPhase(ns, bindingName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, bindingName), TestTimeoutShort).Should(ContainSubstring(message))

			Eventually(IntegrationPod(ns, bindingName), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(),
					MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))

			// Clean up
			Expect(Kamel("delete", bindingName, "-n", ns).Execute()).To(Succeed())
			Expect(DeleteKits(ns)).To(Succeed())
		})
	})
}
