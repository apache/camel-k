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
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuildDontRun(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("deploy")
		t.Run("build and dont run integration", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml",
				"--name", name,
				"--dont-run-after-build",
			).Execute()).To(Succeed())
			// The integration should not change phase until the user request it
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseBuildComplete))
			g.Consistently(IntegrationPhase(t, ctx, ns, name), 10*time.Second).Should(Equal(v1.IntegrationPhaseBuildComplete))
			g.Eventually(Deployment(t, ctx, ns, name)).Should(BeNil())
		})
		t.Run("deploy the integration", func(t *testing.T) {
			g.Expect(Kamel(t, ctx, "deploy", name, "-n", ns).Execute()).To(Succeed())
			// The integration should run immediately
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(Deployment(t, ctx, ns, name)).ShouldNot(BeNil())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("Magicstring!"))
		})
		t.Run("undeploy the integration", func(t *testing.T) {
			g.Expect(Kamel(t, ctx, "undeploy", name, "-n", ns).Execute()).To(Succeed())
			// The integration should change phase suddenly and the resources associated cleared
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseBuildComplete))
			g.Eventually(IntegrationPodsNumbers(t, ctx, ns, name)).Should(Equal(ptr.To(int32(0))))
			g.Eventually(Deployment(t, ctx, ns, name)).Should(BeNil())
		})
	})
}

func TestPipeBuildDontRun(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("pipe-deploy")
		t.Run("build and dont run pipe", func(t *testing.T) {
			g.Expect(KamelBind(t, ctx, ns, "timer-source?message=HelloPipe", "log-sink",
				"--name", name,
				"--annotation", "camel.apache.org/dont-run-after-build=true",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseBuildComplete))
			g.Eventually(PipePhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.PipePhaseBuildComplete))
			g.Consistently(IntegrationPhase(t, ctx, ns, name), 10*time.Second).Should(Equal(v1.IntegrationPhaseBuildComplete))
			g.Consistently(PipePhase(t, ctx, ns, name), 10*time.Second).Should(Equal(v1.PipePhaseBuildComplete))
			g.Eventually(Deployment(t, ctx, ns, name)).Should(BeNil())
			// Pipe condition should indicate build is complete
			g.Eventually(PipeCondition(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(
				WithTransform(PipeConditionReason, Equal("BuildComplete")))
		})
	})
}
