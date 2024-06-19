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

package cli

import (
	"context"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKamelCLIDelete(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("delete running integration", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(Kamel(t, ctx, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete building integration", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(Kamel(t, ctx, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete several integrations", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(Kamel(t, ctx, "delete", "yaml", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
			g.Expect(Kamel(t, ctx, "delete", "java", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "java")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "java"), TestTimeoutLong).Should(BeNil())
		})

		t.Run("delete all integrations", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Eventually(Integration(t, ctx, ns, "yaml")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "yaml"), TestTimeoutLong).Should(BeNil())
			g.Eventually(Integration(t, ctx, ns, "java")).Should(BeNil())
			g.Eventually(IntegrationPod(t, ctx, ns, "java"), TestTimeoutLong).Should(BeNil())
		})
	})
}
