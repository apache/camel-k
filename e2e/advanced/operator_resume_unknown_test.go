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
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestOperatorResumeFromUnknown(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("yaml")
		containerRegistry, ok := os.LookupEnv("KAMEL_INSTALL_REGISTRY")
		g.Expect(ok).To(BeTrue(), "You need to provide the registry as KAMEL_INSTALL_REGISTRY env var")

		InstallOperator(t, ctx, g, ns)
		g.Eventually(OperatorPod(t, ctx, ns)).Should(Not(BeNil()))
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.IntegrationPlatformPhaseReady))
		g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionPlatformAvailable), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Delete the IntegrationPlatform: as soon as there is a "monitoring" operation, the Integration
		// should go in Unknown status as it cannot create traits, therefore, it would fail
		g.Expect(DeletePlatform(t, ctx, ns)()).To(BeTrue())
		g.Consistently(IntegrationPhase(t, ctx, ns, name), 1*time.Minute, 5*time.Second).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		// Asking for a scale operation triggers a monitoring action
		g.Expect(ScaleIntegration(t, ctx, ns, name, 2)).To(Succeed())
		g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseUnknown))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionPlatformAvailable), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		// Fix the platform (create a new one)
		platform := &v1.IntegrationPlatform{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.IntegrationPlatformKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      "camel-k",
			},
			Spec: v1.IntegrationPlatformSpec{
				Build: v1.IntegrationPlatformBuildSpec{
					Registry: v1.RegistrySpec{
						Address: containerRegistry,
					},
				},
			},
		}
		g.Expect(CreateIntegrationPlatform(t, ctx, platform)).To(Succeed())
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.IntegrationPlatformPhaseReady))
		// The monitoring should now start correctly
		g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionPlatformAvailable), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(2))
	})
}
