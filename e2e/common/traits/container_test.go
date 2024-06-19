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

package traits

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestContainerTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Container image pull policy and resources configuration", func(t *testing.T) {
			name := RandomizedSuffixName("java1")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "-t", "container.image-pull-policy=Always", "-t", "container.request-cpu=0.005", "-t", "container.request-memory=100Mi", "-t", "container.limit-cpu=200m", "-t", "container.limit-memory=500Mi", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationPodHas(t, ctx, ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				imagePullPolicy := pod.Spec.Containers[0].ImagePullPolicy
				return imagePullPolicy == "Always"
			}), TestTimeoutShort).Should(BeTrue())
			g.Eventually(IntegrationPodHas(t, ctx, ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				limitsCpu := pod.Spec.Containers[0].Resources.Limits.Cpu()
				requestsCpu := pod.Spec.Containers[0].Resources.Requests.Cpu()
				return limitsCpu != nil && limitsCpu.String() == "200m" && requestsCpu != nil && requestsCpu.String() == "5m"
			}), TestTimeoutShort).Should(BeTrue())
			g.Eventually(IntegrationPodHas(t, ctx, ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				limitsMemory := pod.Spec.Containers[0].Resources.Limits.Memory()
				requestsMemory := pod.Spec.Containers[0].Resources.Requests.Memory()
				return limitsMemory != nil && limitsMemory.String() == "500Mi" && requestsMemory != nil && requestsMemory.String() == "100Mi"
			}), TestTimeoutShort).Should(BeTrue())

		})

		t.Run("Container name", func(t *testing.T) {
			name := RandomizedSuffixName("java2")
			containerName := "my-container-name"
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "-t", "container.name="+containerName, "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationPodHas(t, ctx, ns, name, func(pod *corev1.Pod) bool {
				if len(pod.Spec.Containers) != 1 {
					return false
				}
				podContainerName := pod.Spec.Containers[0].Name
				return podContainerName == containerName
			}), TestTimeoutShort).Should(BeTrue())

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, name)()
			containerTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "container")
			g.Expect(containerTrait).ToNot(BeNil())
			g.Expect(len(containerTrait)).To(Equal(1))
			g.Expect(containerTrait["name"]).To(Equal(containerName))
		})
	})
}
