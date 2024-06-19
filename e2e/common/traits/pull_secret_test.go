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
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

func TestPullSecretTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ocp, err := openshift.IsOpenShift(TestClient(t))
		g.Expect(err).To(BeNil())

		t.Run("Image pull secret is set on pod", func(t *testing.T) {
			name := RandomizedSuffixName("java1")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "pull-secret.enabled=true", "-t", "pull-secret.secret-name=dummy-secret").Execute()).To(Succeed())
			// pod may not run because the pull secret is dummy
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Or(Equal(corev1.PodRunning), Equal(corev1.PodPending)))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
			g.Expect(pod.Spec.ImagePullSecrets[0].Name).To(Equal("dummy-secret"))
		})

		t.Run("Explicitly disable image pull secret", func(t *testing.T) {
			name := RandomizedSuffixName("java2")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "pull-secret.enabled=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, name)()
			pullSecretTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "pull-secret")
			g.Expect(pullSecretTrait).ToNot(BeNil())
			g.Expect(len(pullSecretTrait)).To(Equal(1))
			g.Expect(pullSecretTrait["enabled"]).To(Equal(false))

			pod := IntegrationPod(t, ctx, ns, name)()
			if ocp {
				// OpenShift `default` service account has imagePullSecrets so it's always set
				g.Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
			} else {
				g.Expect(pod.Spec.ImagePullSecrets).To(BeNil())
			}
		})

		if ocp {
			// OpenShift always has an internal registry so image pull secret is set by default
			t.Run("Image pull secret is automatically set by default", func(t *testing.T) {
				name := RandomizedSuffixName("java3")
				g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())
				g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

				pod := IntegrationPod(t, ctx, ns, name)()
				g.Expect(pod.Spec.ImagePullSecrets).NotTo(BeEmpty())
				g.Expect(pod.Spec.ImagePullSecrets[0].Name).To(HavePrefix("default-dockercfg-"))
			})
		}
	})
}
