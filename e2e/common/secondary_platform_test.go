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
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestSecondaryPlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(ConfigureSecondayPlatfromWith(ns, func(p *v1.IntegrationPlatform) {
			p.Name = "secondary"
			if p.Spec.Traits == nil {
				p.Spec.Traits = make(map[string]v1.TraitSpec)
			}
			p.Spec.Traits["container"] = v1.TraitSpec{
				Configuration: AsTraitConfiguration(map[string]string{
					"limitCPU": "0.1",
				}),
			}
		})).To(Succeed())

		Expect(Kamel("run", "-n", ns, "--name", "limited", "--annotation", "camel.apache.org/platform.id=secondary", "files/yaml.yaml").Execute()).To(Succeed())

		Eventually(IntegrationPod(ns, "limited"), TestTimeoutMedium).Should(Not(BeNil()))
		Eventually(IntegrationPodHas(ns, "limited", func(pod *corev1.Pod) bool {
			if len(pod.Spec.Containers) != 1 {
				return false
			}
			cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
			return cpuLimits != nil && cpuLimits.AsApproximateFloat64() > 0
		}), TestTimeoutShort).Should(BeTrue())
		Expect(Kamel("delete", "limited", "-n", ns).Execute()).To(Succeed())

		Expect(Kamel("run", "-n", ns, "--name", "normal", "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPod(ns, "normal"), TestTimeoutShort).Should(Not(BeNil()))
		Eventually(IntegrationPodHas(ns, "normal", func(pod *corev1.Pod) bool {
			if len(pod.Spec.Containers) != 1 {
				return false
			}
			cpuLimits := pod.Spec.Containers[0].Resources.Limits.Cpu()
			return cpuLimits == nil || cpuLimits.IsZero()
		}), TestTimeoutShort).Should(BeTrue())

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
