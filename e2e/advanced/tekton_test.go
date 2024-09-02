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

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

// TestTektonLikeBehavior verifies that the kamel binary can be invoked from within the Camel K image.
// This feature is used in Tekton pipelines.
func TestTektonLikeBehavior(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperator(t, ctx, g, ns)

		// Store a configmap holding an integration source
		var cmData = make(map[string][]byte)
		source, err := os.ReadFile("./files/Java.java")
		require.NoError(t, err)
		cmData["Java.java"] = source
		err = CreateBinaryConfigmap(t, ctx, ns, "integration-file", cmData)
		require.NoError(t, err)
		integration := v1.ValueSource{
			ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
				LocalObjectReference: corev1.LocalObjectReference{
					Name: "integration-file",
				},
				Key: "Java.java",
			},
		}
		t.Run("Run tekton task like delegate build and run to operator", func(t *testing.T) {
			name := RandomizedSuffixName("java-tekton-basic")
			g.Expect(CreateKamelPodWithIntegrationSource(t, ctx, ns, "tekton-task-basic", integration, "run", "/tmp/Java.java", "--name", name)).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})
		t.Run("Run tekton task like delegate run to operator", func(t *testing.T) {
			name := RandomizedSuffixName("java-tekton-run")
			// Use an external image as source
			externalImage := "quay.io/fuse_qe/echo-server:0.3.3"
			g.Expect(CreateKamelPodWithIntegrationSource(t, ctx, ns, "tekton-task-run", integration, "run", "/tmp/Java.java", "-t", "container.image="+externalImage, "--name", name)).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Echo"))
			tektonIntegrationKitName := IntegrationKit(t, ctx, ns, name)()
			g.Expect(tektonIntegrationKitName).ToNot(BeEmpty())
			tektonIntegrationKitImage := KitImage(t, ctx, ns, tektonIntegrationKitName)()
			g.Expect(tektonIntegrationKitImage).ToNot(BeEmpty())
			g.Eventually(Kit(t, ctx, ns, tektonIntegrationKitName)().Status.Image, TestTimeoutShort).Should(Equal(externalImage))
		})
	})
}
