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

func TestPipeCrossNamespaceKamelet(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns1 string) {
		t.Run("store kamelet", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "cross-ns/my-timer-source-ns.kamelet.yaml", "-n", ns1))
		})

		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns2 string) {
			t.Run("set privileges", func(t *testing.T) {
				ExpectExecSucceed(t, g, Kubectl("apply", "-f", "cross-ns/sa.yaml", "-n", ns2))
				ExpectExecSucceed(t, g, Kubectl("apply", "-f", "cross-ns/sa-role.yaml", "-n", ns1))
				saRbFile := cloneAndReplaceNamespace(t, "cross-ns/sa-rolebinding.yaml", ns2)
				ExpectExecSucceed(t, g, Kubectl("apply", "-f", saRbFile, "-n", ns1))
			})
			t.Run("cross namespace pipe", func(t *testing.T) {
				// Clone the resource in a temporary file as it will require to be changed
				pipeFile := cloneAndReplaceNamespace(t, "cross-ns/pipe-cross-ns.yaml", ns1)
				name := "pipe-cross-ns"
				ExpectExecSucceed(t, g, Kubectl("apply", "-f", pipeFile, "-n", ns2))
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
