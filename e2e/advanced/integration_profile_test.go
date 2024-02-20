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
	"testing"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

func TestIntegrationProfile(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-integration-profile"
		Expect(KamelInstallWithID(operatorID, ns, "--global", "--force").Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		integrationProfile := v1.NewIntegrationProfile(ns, "ipr-global")
		integrationProfile.SetOperatorID(operatorID)
		integrationProfile.Spec.Traits.Container = &traitv1.ContainerTrait{
			Name:     "ck-integration-global",
			LimitCPU: "0.2",
		}

		Expect(CreateIntegrationProfile(&integrationProfile)).To(Succeed())
		Eventually(SelectedIntegrationProfilePhase(ns, "ipr-global"), TestTimeoutMedium).Should(Equal(v1.IntegrationProfilePhaseReady))

		WithNewTestNamespace(t, func(ns1 string) {
			integrationProfile := v1.NewIntegrationProfile(ns1, "ipr-local")
			integrationProfile.SetOperatorID(operatorID)
			integrationProfile.Spec.Traits.Container = &traitv1.ContainerTrait{
				LimitCPU: "0.1",
			}
			Expect(CreateIntegrationProfile(&integrationProfile)).To(Succeed())
			Eventually(SelectedIntegrationProfilePhase(ns1, "ipr-local"), TestTimeoutMedium).Should(Equal(v1.IntegrationProfilePhaseReady))

			t.Run("Run integration with global integration profile", func(t *testing.T) {
				Expect(KamelRunWithID(operatorID, ns1, "--name", "limited", "--integration-profile", "ipr-global", "files/yaml.yaml").Execute()).To(Succeed())

				Eventually(IntegrationPod(ns1, "limited"), TestTimeoutMedium).Should(Not(BeNil()))
				Eventually(IntegrationPodHas(ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					containerName := pod.Spec.Containers[0].Name
					return containerName == "ck-integration-global"
				}), TestTimeoutShort).Should(BeTrue())
				Eventually(IntegrationPodHas(ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
					return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
				}), TestTimeoutShort).Should(BeTrue())
				Expect(Kamel("delete", "limited", "-n", ns1).Execute()).To(Succeed())
			})

			t.Run("Run integration with namespace local integration profile", func(t *testing.T) {
				Expect(KamelRunWithID(operatorID, ns1, "--name", "limited", "--integration-profile", "ipr-local", "files/yaml.yaml").Execute()).To(Succeed())

				Eventually(IntegrationPod(ns1, "limited"), TestTimeoutMedium).Should(Not(BeNil()))
				Eventually(IntegrationPodHas(ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					containerName := pod.Spec.Containers[0].Name
					return containerName == "integration"
				}), TestTimeoutShort).Should(BeTrue())

				Eventually(IntegrationPodHas(ns1, "limited", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
					return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
				}), TestTimeoutShort).Should(BeTrue())
				Expect(Kamel("delete", "limited", "-n", ns1).Execute()).To(Succeed())
			})

			t.Run("Run integration without integration profile", func(t *testing.T) {
				Expect(KamelRunWithID(operatorID, ns1, "--name", "normal", "files/yaml.yaml").Execute()).To(Succeed())
				Eventually(IntegrationPod(ns1, "normal"), TestTimeoutShort).Should(Not(BeNil()))
				Eventually(IntegrationPodHas(ns1, "normal", func(pod *corev1.Pod) bool {
					if len(pod.Spec.Containers) != 1 {
						return false
					}
					cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
					return cpuLimits == nil || cpuLimits.IsZero()
				}), TestTimeoutShort).Should(BeTrue())
			})

			// Clean up
			Expect(Kamel("delete", "--all", "-n", ns1).Execute()).To(Succeed())
		})
	})
}

func TestIntegrationProfileInfluencesKit(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-ipr-kit"
		Expect(KamelInstallWithID(operatorID, ns, "--global", "--force").Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		integrationProfile := v1.NewIntegrationProfile(ns, "ipr-global")
		integrationProfile.SetOperatorID(operatorID)
		integrationProfile.Spec.Traits.Builder = &traitv1.BuilderTrait{
			Properties: []string{"b1=foo"},
		}

		Expect(CreateIntegrationProfile(&integrationProfile)).To(Succeed())
		Eventually(SelectedIntegrationProfilePhase(ns, "ipr-global"), TestTimeoutMedium).Should(Equal(v1.IntegrationProfilePhaseReady))

		Expect(KamelRunWithID(operatorID, ns, "--name", "normal", "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPod(ns, "normal"), TestTimeoutMedium).Should(Not(BeNil()))
		Eventually(IntegrationPodPhase(ns, "normal"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "normal", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "normal"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		// Verify that a new kit has been built based on the default base image
		integrationKitName := IntegrationKit(ns, "normal")()
		Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		Eventually(Kit(ns, integrationKitName)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		Expect(KamelRunWithID(operatorID, ns, "--name", "simple", "--integration-profile", "ipr-global", "files/yaml.yaml").Execute()).To(Succeed())

		Eventually(IntegrationPod(ns, "simple"), TestTimeoutMedium).Should(Not(BeNil()))
		Eventually(IntegrationPodPhase(ns, "simple"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "simple", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "simple"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Verify that a new kit has been built based on the previous kit
		integrationKitNameWithProfile := IntegrationKit(ns, "simple")()
		Eventually(integrationKitNameWithProfile).ShouldNot(Equal(integrationKitName))
		Eventually(Kit(ns, integrationKitNameWithProfile)().Status.BaseImage).Should(ContainSubstring(integrationKitName))
		Eventually(Kit(ns, integrationKitNameWithProfile)().Status.RootImage).Should(Equal(defaults.BaseImage()))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestPropagateIntegrationProfileChanges(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-ipr-changes"
		Expect(KamelInstallWithID(operatorID, ns, "--global", "--force").Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		integrationProfile := v1.NewIntegrationProfile(ns, "debug-profile")
		integrationProfile.SetOperatorID(operatorID)
		integrationProfile.Spec.Traits.Container = &traitv1.ContainerTrait{
			Name: "ck-ipr",
		}
		integrationProfile.Spec.Traits.Logging = &traitv1.LoggingTrait{
			Level: "DEBUG",
		}

		Expect(CreateIntegrationProfile(&integrationProfile)).To(Succeed())
		Eventually(SelectedIntegrationProfilePhase(ns, "debug-profile"), TestTimeoutMedium).Should(Equal(v1.IntegrationProfilePhaseReady))

		Expect(KamelRunWithID(operatorID, ns, "--name", "simple", "--integration-profile", "debug-profile", "files/yaml.yaml").Execute()).To(Succeed())

		Eventually(IntegrationPod(ns, "simple"), TestTimeoutMedium).Should(Not(BeNil()))
		Eventually(IntegrationPodHas(ns, "simple", func(pod *corev1.Pod) bool {
			if len(pod.Spec.Containers) != 1 {
				return false
			}
			containerName := pod.Spec.Containers[0].Name
			return containerName == "ck-ipr"
		}), TestTimeoutShort).Should(BeTrue())

		Expect(UpdateIntegrationProfile(ns, func(ipr *v1.IntegrationProfile) {
			ipr.Spec.Traits.Container = &traitv1.ContainerTrait{
				Name: "ck-ipr-new",
			}
		})).To(Succeed())

		Eventually(IntegrationPodHas(ns, "simple", func(pod *corev1.Pod) bool {
			if len(pod.Spec.Containers) != 1 {
				return false
			}
			containerName := pod.Spec.Containers[0].Name
			return containerName == "ck-ipr-new"
		}), TestTimeoutShort).Should(BeTrue())

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
