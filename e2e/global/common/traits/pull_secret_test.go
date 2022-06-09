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

package traits

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestPullSecretTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		ocp, err := openshift.IsOpenShift(TestClient())
		Expect(err).To(BeNil())

		operatorID := "camel-k-trait-pull-secret"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("Image pull secret is set on pod", func(t *testing.T) {
			name := "java1"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
				"-t", "pull-secret.enabled=true",
				"-t", "pull-secret.secret-name=dummy-secret").Execute()).To(Succeed())
			// pod may not run because the pull secret is dummy
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Or(Equal(corev1.PodRunning), Equal(corev1.PodPending)))

			pod := IntegrationPod(ns, name)()
			Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
			Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("dummy-secret"))
		})

		t.Run("Explicitly disable image pull secret", func(t *testing.T) {
			name := "java2"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name,
				"-t", "pull-secret.enabled=false").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, name)()
			if ocp {
				// OpenShift `default` service account has imagePullSecrets so it's always set
				Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
			} else {
				Expect(pod.Spec.ImagePullSecrets).To(BeNil())
			}
		})

		if ocp {
			// OpenShift always has an internal registry so image pull secret is set by default
			t.Run("Image pull secret is automatically set by default", func(t *testing.T) {
				name := "java3"
				Expect(KamelRunWithID(operatorID, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
				Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

				pod := IntegrationPod(ns, name)()
				Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
				Expect(pod.Spec.ImagePullSecrets[0].Name).To(HavePrefix("default-dockercfg-"))
			})
		}

		// Clean-up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
