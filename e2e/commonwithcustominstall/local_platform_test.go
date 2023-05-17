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

package commonwithcustominstall

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

func TestLocalPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-platform-local"
		Expect(KamelInstallWithID(operatorID, ns, "--global", "--force").Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		pl := Platform(ns)()
		pl.Spec.Pipeline.Maven.Properties = make(map[string]string)
		pl.Spec.Pipeline.Maven.Properties["build-global-prop1"] = "build-global-value1"
		// set maximum number of running builds
		pl.Spec.Pipeline.MaxRunningPipelines = 1
		if err := TestClient().Update(TestContext, pl); err != nil {
			t.Error(err)
			t.FailNow()
		}

		Eventually(PlatformHas(ns, func(pl *v1.IntegrationPlatform) bool {
			return pl.Status.Pipeline.MaxRunningPipelines == 1
		}), TestTimeoutMedium).Should(BeTrue())

		WithNewTestNamespace(t, func(ns1 string) {
			localPlatform := v1.NewIntegrationPlatform(ns1, operatorID)
			localPlatform.Spec.Pipeline.Maven.Properties = make(map[string]string)
			localPlatform.Spec.Pipeline.Maven.Properties["build-local-prop1"] = "build-local-value1"
			localPlatform.SetOperatorID(operatorID)

			localPlatform.Spec.Traits.Container = &traitv1.ContainerTrait{
				LimitCPU: "0.1",
			}

			if err := TestClient().Create(TestContext, &localPlatform); err != nil {
				t.Error(err)
				t.FailNow()
			}
			Eventually(PlatformPhase(ns1), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
			Eventually(PlatformHas(ns1, func(pl *v1.IntegrationPlatform) bool {
				return pl.Status.Cluster != ""
			}), TestTimeoutShort).Should(BeTrue())

			Eventually(PlatformHas(ns1, func(pl *v1.IntegrationPlatform) bool {
				return pl.Status.Pipeline.MaxRunningPipelines == 1
			}), TestTimeoutShort).Should(BeTrue())

			local := Platform(ns1)()
			Expect(local.Status.Pipeline.PublishStrategy).To(Equal(pl.Status.Pipeline.PublishStrategy))
			Expect(local.Status.Pipeline.BuildConfiguration.Strategy).To(Equal(pl.Status.Pipeline.BuildConfiguration.Strategy))
			Expect(local.Status.Pipeline.Maven.LocalRepository).To(Equal(pl.Status.Pipeline.Maven.LocalRepository))
			Expect(local.Status.Pipeline.Maven.CLIOptions).To(ContainElements(pl.Status.Pipeline.Maven.CLIOptions))
			Expect(local.Status.Pipeline.Maven.Extension).To(BeEmpty())
			Expect(local.Status.Pipeline.Maven.Properties).To(HaveLen(2))
			Expect(local.Status.Pipeline.Maven.Properties["build-global-prop1"]).To(Equal("build-global-value1"))
			Expect(local.Status.Pipeline.Maven.Properties["build-local-prop1"]).To(Equal("build-local-value1"))

			Expect(KamelRunWithID(operatorID, ns1, "--name", "local-integration", "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPod(ns1, "local-integration"), TestTimeoutMedium).Should(Not(BeNil()))
			Eventually(IntegrationPodHas(ns1, "local-integration", func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
				return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
			}), TestTimeoutShort).Should(BeTrue())

			// Clean up
			Expect(Kamel("delete", "--all", "-n", ns1).Execute()).To(Succeed())
		})
	})
}
