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
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKamelPromoteGitOps(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("build and run gitops", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady)).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})
		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsTarget string) {
			// Export to GitOps directory structure
			tmpDir, err := os.MkdirTemp("", "ck-promote-it-*")
			if err != nil {
				t.Error(err)
			}
			g.Expect(Kamel(t, ctx, "promote", "yaml", "-n", ns, "--to", nsTarget, "--export-gitops-dir", tmpDir).Execute()).To(Succeed())
			// Run the exported Integration as it would be any CICD
			ExpectExecSucceed(t, g, Kubectl("apply", "-k", tmpDir+"/yaml/overlays/"+nsTarget))
			g.Eventually(IntegrationPodPhase(t, ctx, nsTarget, "yaml"), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, nsTarget, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, nsTarget, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// Make sure that no IntegrationKit was ever built for this Integration
			g.Eventually(IntegrationKit(t, ctx, nsTarget, "yaml")).Should(Equal(""))
		})
	})
}
