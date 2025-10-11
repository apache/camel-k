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
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKameletDiscoverCapabilities(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("discover kamelet capabilities", func(t *testing.T) {
			template := map[string]interface{}{
				"from": map[string]interface{}{
					"uri": "platform-http:///webhook",
					"steps": []map[string]interface{}{
						{
							"to": "kamelet:sink",
						},
					},
				},
			}
			g.Expect(CreateKamelet(t, ctx, ns, "capabilities-webhook-source", template, nil, nil)()).To(Succeed())

			name := RandomizedSuffixName("webhook")
			g.Expect(KamelRun(t, ctx, ns, "files/webhook.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Started route1 (kamelet://capabilities-webhook-source)"))
			// Verify Integration capabilities
			g.Eventually(IntegrationStatusCapabilities(t, ctx, ns, name), TestTimeoutShort).Should(ContainElements("platform-http"))
			// Verify expected resources from Kamelet (Service in this case)
			service := Service(t, ctx, ns, name)
			g.Eventually(service, TestTimeoutShort).ShouldNot(BeNil())
		})
	})
}

func TestKameletMultiVersions(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("multiple kamelet versions", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/my-timer-source.kamelet.yaml", "-n", ns))
			name := RandomizedSuffixName("multiversions")
			g.Expect(KamelRun(t, ctx, ns, "files/kamelet-it-main.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Kamelet Main"))
			// Switch to Kamelet V1
			g.Expect(KamelRun(t, ctx, ns, "files/kamelet-it-v1.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Kamelet V1"))
		})
	})
}

func TestKameletNamespaced(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns1 string) {
		t.Run("store kamelet", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/my-timer-source-ns.kamelet.yaml", "-n", ns1))
		})

		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns2 string) {
			t.Run("namespaced kamelet", func(t *testing.T) {
				// Clone the resource in a temporary file as it will require to be changed
				routeFile := cloneAndReplaceNamespace(t, "files/kamelet-it-ns.yaml", ns1)
				name := RandomizedSuffixName("namespaced-kamelet")
				g.Expect(KamelRun(t, ctx, ns2, routeFile, "--name", name).Execute()).To(Succeed())
				g.Eventually(IntegrationConditionStatus(t, ctx, ns2, name, v1.IntegrationConditionReady), TestTimeoutMedium).
					Should(Equal(corev1.ConditionTrue))
				g.Eventually(IntegrationPodPhase(t, ctx, ns2, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns2, name), TestTimeoutShort).Should(ContainSubstring("Kamelet NS"))
			})
		})
	})
}

// cloneAndReplaceNamespace clones and replace the content marked as %%% with the namespace passed as parameter.
func cloneAndReplaceNamespace(t *testing.T, srcPath, namespace string) string {
	t.Helper()

	tempDir := t.TempDir()
	tempPath := filepath.Join(tempDir, filepath.Base(srcPath))

	dstFile, err := os.Create(tempPath)
	if err != nil {
		t.Fatalf("failed to create temp file: %v", err)
	}
	defer dstFile.Close()

	content, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("failed to read src file: %v", err)
	}
	updated := strings.ReplaceAll(string(content), "%%%", namespace)
	err = os.WriteFile(tempPath, []byte(updated), 0644)
	if err != nil {
		t.Fatalf("failed to write dst file: %v", err)
	}

	return tempPath
}
