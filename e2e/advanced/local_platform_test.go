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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

func TestLocalPlatform(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-platform-local"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--global", "--force")).To(Succeed())
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		g.Expect(UpdatePlatform(t, ctx, ns, operatorID, func(pl *v1.IntegrationPlatform) {
			pl.Spec.Build.Maven.Properties = make(map[string]string)
			pl.Spec.Build.Maven.Properties["build-global-prop1"] = "build-global-value1"
			// set maximum number of running builds
			pl.Spec.Build.MaxRunningBuilds = 1
		})).To(Succeed())

		g.Eventually(PlatformHas(t, ctx, ns, func(pl *v1.IntegrationPlatform) bool {
			return pl.Status.Build.MaxRunningBuilds == 1
		}), TestTimeoutMedium).Should(BeTrue())

		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns1 string) {
			// Install platform (use the installer to get staging if present)
			g.Expect(KamelInstallWithID(t, ctx, "local-platform", ns1, "--skip-operator-setup")).To(Succeed())

			g.Expect(UpdatePlatform(t, ctx, ns1, "local-platform", func(pl *v1.IntegrationPlatform) {
				pl.Spec.Build.Maven.Properties = make(map[string]string)
				pl.Spec.Build.Maven.Properties["build-local-prop1"] = "build-local-value1"
				pl.SetOperatorID(operatorID)

				pl.Spec.Traits.Container = &traitv1.ContainerTrait{
					LimitCPU: "0.1",
				}
			})).To(Succeed())

			g.Eventually(PlatformPhase(t, ctx, ns1), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
			g.Eventually(PlatformHas(t, ctx, ns1, func(pl *v1.IntegrationPlatform) bool {
				return pl.Status.Cluster != ""
			}), TestTimeoutShort).Should(BeTrue())

			g.Eventually(PlatformHas(t, ctx, ns1, func(pl *v1.IntegrationPlatform) bool {
				return pl.Status.Build.MaxRunningBuilds == 1
			}), TestTimeoutShort).Should(BeTrue())

			pl := PlatformByName(t, ctx, ns, operatorID)()
			local := Platform(t, ctx, ns1)()
			g.Expect(local.Status.Build.PublishStrategy).To(Equal(pl.Status.Build.PublishStrategy))
			g.Expect(local.Status.Build.BuildConfiguration.Strategy).To(Equal(pl.Status.Build.BuildConfiguration.Strategy))
			g.Expect(local.Status.Build.BuildConfiguration.OrderStrategy).To(Equal(pl.Status.Build.BuildConfiguration.OrderStrategy))
			g.Expect(local.Status.Build.Maven.LocalRepository).To(Equal(pl.Status.Build.Maven.LocalRepository))
			g.Expect(local.Status.Build.Maven.CLIOptions).To(ContainElements(pl.Status.Build.Maven.CLIOptions))
			g.Expect(local.Status.Build.Maven.Extension).To(BeEmpty())
			g.Expect(local.Status.Build.Maven.Properties).To(HaveLen(2))
			g.Expect(local.Status.Build.Maven.Properties["build-global-prop1"]).To(Equal("build-global-value1"))
			g.Expect(local.Status.Build.Maven.Properties["build-local-prop1"]).To(Equal("build-local-value1"))

			g.Expect(CamelKRunWithID(t, ctx, operatorID, ns1, "--name", "local-integration", "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPod(t, ctx, ns1, "local-integration"), TestTimeoutMedium).Should(Not(BeNil()))
			g.Eventually(IntegrationPodHas(t, ctx, ns1, "local-integration", func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
				return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
			}), TestTimeoutShort).Should(BeTrue())

			// Clean up
			g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns1).Execute()).To(Succeed())
		})
	})
}
