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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestNativeBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns,
			"--build-timeout", "15m0s",
			"--operator-resources", "limits.memory=4Gi",
		).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("kamelet binding with native build", func(t *testing.T) {
			from := corev1.ObjectReference{
				Kind:       "Kamelet",
				Name:       "timer-source",
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			}

			to := corev1.ObjectReference{
				Kind:       "Kamelet",
				Name:       "log-sink",
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			}

			bindingName := "native-binding"
			message := "Magicstring!"
			Expect(BindKameletTo(ns, bindingName, map[string]string{"trait.camel.apache.org/quarkus.package-type": "native"}, from, to, map[string]string{"message": message}, map[string]string{})()).To(Succeed())

			Eventually(Kits(ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutLong).Should(HaveLen(1))
			nativeKit := Kits(ns, withNativeLayout, KitWithPhase(v1.IntegrationKitPhaseReady))()[0]
			Eventually(IntegrationKit(ns, bindingName), TestTimeoutShort).Should(Equal(nativeKit.Name))

			Eventually(IntegrationPodPhase(ns, bindingName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, bindingName), TestTimeoutShort).Should(ContainSubstring(message))

			Eventually(IntegrationPod(ns, bindingName), TestTimeoutShort).
				Should(WithTransform(getContainerCommand(), MatchRegexp(".*camel-k-integration-\\d+\\.\\d+\\.\\d+[-A-Za-z]*-runner.*")))

			// Clean up
			Expect(Kamel("delete", bindingName, "-n", ns).Execute()).To(Succeed())
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
