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

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

func TestIntegrationProfile(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-integration-profile"
		InstallOperatorWithConf(t, ctx, g, ns, operatorID, true, nil)

		integrationProfile := v1.NewIntegrationProfile(ns, "ipr-global")
		integrationProfile.SetOperatorID(operatorID)
		integrationProfile.Spec.Traits.Container = &traitv1.ContainerTrait{
			Name:     "ck-integration-global",
			LimitCPU: "0.3",
		}

		g.Expect(CreateIntegrationProfile(t, ctx, &integrationProfile)).To(Succeed())

		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns1 string) {
			integrationProfile := v1.NewIntegrationProfile(ns1, "ipr-local")
			integrationProfile.SetOperatorID(operatorID)
			integrationProfile.Spec.Traits.Container = &traitv1.ContainerTrait{
				LimitCPU: "0.2",
			}
			g.Expect(CreateIntegrationProfile(t, ctx, &integrationProfile)).To(Succeed())

			t.Run("Run integration with global integration profile", func(t *testing.T) {
				g.Expect(KamelRunWithID(t, ctx, operatorID, ns1, "--name", "limited", "--integration-profile", "ipr-global", "files/yaml.yaml").Execute()).To(Succeed())

				g.Eventually(IntegrationPod(t, ctx, ns1, "limited"), TestTimeoutMedium).Should(Not(BeNil()))
				g.Eventually(IntegrationPodHas(t, ctx, ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					containerName := pod.Spec.Containers[0].Name
					return containerName == "ck-integration-global"
				}), TestTimeoutShort).Should(BeTrue())
				g.Eventually(IntegrationPodHas(t, ctx, ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
					return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
				}), TestTimeoutShort).Should(BeTrue())
				g.Expect(Kamel(t, ctx, "delete", "limited", "-n", ns1).Execute()).To(Succeed())
			})

			t.Run("Run integration with namespace local integration profile", func(t *testing.T) {
				g.Expect(KamelRunWithID(t, ctx, operatorID, ns1, "--name", "limited", "--integration-profile", "ipr-local", "files/yaml.yaml").Execute()).To(Succeed())

				g.Eventually(IntegrationPod(t, ctx, ns1, "limited"), TestTimeoutMedium).Should(Not(BeNil()))
				g.Eventually(IntegrationPodHas(t, ctx, ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					containerName := pod.Spec.Containers[0].Name
					return containerName == "integration"
				}), TestTimeoutShort).Should(BeTrue())

				g.Eventually(IntegrationPodHas(t, ctx, ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
					return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
				}), TestTimeoutShort).Should(BeTrue())
				g.Expect(Kamel(t, ctx, "delete", "limited", "-n", ns1).Execute()).To(Succeed())
			})
		})
	})
}

func TestIntegrationProfileInfluencesKit(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-ipr-kit"
		InstallOperatorWithConf(t, ctx, g, ns, operatorID, false, nil)

		integrationProfile := v1.NewIntegrationProfile(ns, "ipr-global")
		integrationProfile.SetOperatorID(operatorID)
		integrationProfile.Spec.Traits.Builder = &traitv1.BuilderTrait{
			Properties: []string{"b1=foo"},
		}

		g.Expect(CreateIntegrationProfile(t, ctx, &integrationProfile)).To(Succeed())

		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "--name", "normal", "files/yaml.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "normal", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPod(t, ctx, ns, "normal"), TestTimeoutMedium).Should(Not(BeNil()))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "normal"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "normal"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		// Verify that a new kit has been built based on the default base image
		integrationKitName := IntegrationKit(t, ctx, ns, "normal")()
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		g.Eventually(Kit(t, ctx, ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "--name", "simple", "--integration-profile", "ipr-global", "files/yaml.yaml").Execute()).To(Succeed())

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "simple", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPod(t, ctx, ns, "simple"), TestTimeoutMedium).Should(Not(BeNil()))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "simple"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "simple"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Verify that a new kit has been built based on the previous kit
		integrationKitNameWithProfile := IntegrationKit(t, ctx, ns, "simple")()
		g.Eventually(integrationKitNameWithProfile).ShouldNot(Equal(integrationKitName))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameWithProfile)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
		g.Eventually(Kit(t, ctx, ns, integrationKitNameWithProfile)().Status.RootImage).Should(Equal(defaults.BaseImage()))
	})
}

func TestPropagateIntegrationProfileChanges(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-ipr-changes"
		InstallOperatorWithConf(t, ctx, g, ns, operatorID, false, nil)

		integrationProfile := v1.NewIntegrationProfile(ns, "debug-profile")
		integrationProfile.SetOperatorID(operatorID)
		integrationProfile.Spec.Traits.Container = &traitv1.ContainerTrait{
			Name: "ck-ipr",
		}
		integrationProfile.Spec.Traits.Logging = &traitv1.LoggingTrait{
			Level: "DEBUG",
		}

		g.Expect(CreateIntegrationProfile(t, ctx, &integrationProfile)).To(Succeed())
		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "--name", "simple", "--integration-profile", "debug-profile", "files/yaml.yaml").Execute()).To(Succeed())

		g.Eventually(IntegrationPod(t, ctx, ns, "simple"), TestTimeoutMedium).Should(Not(BeNil()))
		g.Eventually(IntegrationPodHas(t, ctx, ns, "simple", func(pod *corev1.Pod) bool {
			if len(pod.Spec.Containers) != 1 {
				return false
			}
			containerName := pod.Spec.Containers[0].Name
			return containerName == "ck-ipr"
		}), TestTimeoutShort).Should(BeTrue())

		g.Expect(UpdateIntegrationProfile(t, ctx, ns, func(ipr *v1.IntegrationProfile) {
			ipr.Spec.Traits.Container = &traitv1.ContainerTrait{
				Name: "ck-ipr-new",
			}
		})).To(Succeed())

		g.Eventually(IntegrationPodHas(t, ctx, ns, "simple", func(pod *corev1.Pod) bool {
			if len(pod.Spec.Containers) != 1 {
				return false
			}
			containerName := pod.Spec.Containers[0].Name
			return containerName == "ck-ipr-new"
		}), TestTimeoutShort).Should(BeTrue())
	})
}
