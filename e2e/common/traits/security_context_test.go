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
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestSecurityContextTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Run with default security context", func(t *testing.T) {
			name := RandomizedSuffixName("java-sc-default")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod.Spec.SecurityContext).NotTo(BeNil())
			g.Expect(pod.Spec.SecurityContext.RunAsUser).To(Equal(ptr.To[int64](1000)))
			g.Expect(pod.Spec.SecurityContext.RunAsNonRoot).To(Equal(ptr.To[bool](true)))
			g.Expect(pod.Spec.SecurityContext.SeccompProfile).NotTo(BeNil())
			g.Expect(pod.Spec.SecurityContext.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeRuntimeDefault))
		})

		t.Run("run with custom secuirty context", func(t *testing.T) {
			name := RandomizedSuffixName("java-sc-custom")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "security-context.run-as-user=2000", "-t", "security-context.seccomp-profile-type=Unconfined").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod.Spec.SecurityContext).NotTo(BeNil())
			g.Expect(pod.Spec.SecurityContext.RunAsUser).To(Equal(ptr.To[int64](2000)))
			g.Expect(pod.Spec.SecurityContext.RunAsNonRoot).To(Equal(ptr.To[bool](true)))
			g.Expect(pod.Spec.SecurityContext.SeccompProfile).NotTo(BeNil())
			g.Expect(pod.Spec.SecurityContext.SeccompProfile.Type).To(Equal(corev1.SeccompProfileTypeUnconfined))
		})
	})
}
