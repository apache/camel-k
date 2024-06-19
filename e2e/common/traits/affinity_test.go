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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestAffinityTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		var hostname string
		if node, err := selectSchedulableNode(t, ctx); err == nil {
			hostname = node.Labels["kubernetes.io/hostname"]
		} else {
			// if 'get nodes' is not allowed, just skip tests for node affinity
			hostname = ""
		}

		t.Run("Run Java with node affinity", func(t *testing.T) {
			if hostname == "" {
				t.Skip(`This test requires node to have "kubernetes.io/hostname" label`)
			}
			name1 := RandomizedSuffixName("java1")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name1,
				"-t", "affinity.enabled=true",
				"-t", fmt.Sprintf("affinity.node-affinity-labels=kubernetes.io/hostname in(%s)", hostname)).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name1)).Should(Equal(corev1.PodRunning))
			g.Eventually(
				IntegrationConditionStatus(t, ctx, ns, name1, v1.IntegrationConditionReady), TestTimeoutShort).Should(
				Equal(corev1.ConditionTrue),
			)
			g.Eventually(IntegrationLogs(t, ctx, ns, name1), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, name1)()
			g.Expect(pod.Spec.Affinity).NotTo(BeNil())
			g.Expect(pod.Spec.Affinity.NodeAffinity).To(Equal(&corev1.NodeAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: nodeSelector("kubernetes.io/hostname", corev1.NodeSelectorOpIn, hostname),
			}))
			g.Expect(pod.Spec.NodeName).To(Equal(hostname))
		})

		t.Run("Run Java with pod affinity", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", "java2",
				"-t", "affinity.enabled=true",
				"-t", "affinity.pod-affinity-labels=camel.apache.org/integration").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java2")).Should(Equal(corev1.PodRunning))
			g.Eventually(
				IntegrationConditionStatus(t, ctx, ns, "java2", v1.IntegrationConditionReady), TestTimeoutShort).Should(
				Equal(corev1.ConditionTrue),
			)
			g.Eventually(IntegrationLogs(t, ctx, ns, "java2"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, "java2")()
			g.Expect(pod.Spec.Affinity).NotTo(BeNil())
			g.Expect(pod.Spec.Affinity.PodAffinity).To(Equal(&corev1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					podAffinityTerm("camel.apache.org/integration", metav1.LabelSelectorOpExists, "kubernetes.io/hostname"),
				},
			}))
		})

		t.Run("Run Java with pod anti affinity", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", "java3",
				"-t", "affinity.enabled=true",
				"-t", "affinity.pod-anti-affinity-labels=camel.apache.org/integration").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java3")).Should(Equal(corev1.PodPending))
			g.Eventually(
				IntegrationConditionStatus(t, ctx, ns, "java3", v1.IntegrationConditionReady), TestTimeoutShort).Should(
				Equal(corev1.ConditionFalse),
			)
			g.Eventually(IntegrationCondition(t, ctx, ns, "java3", v1.IntegrationConditionReady)().Message).
				Should(ContainSubstring("didn't match pod anti-affinity rules"))

			pod := IntegrationPod(t, ctx, ns, "java3")()
			g.Expect(pod.Spec.Affinity).NotTo(BeNil())
			g.Expect(pod.Spec.Affinity.PodAntiAffinity).To(Equal(&corev1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []corev1.PodAffinityTerm{
					podAffinityTerm("camel.apache.org/integration", metav1.LabelSelectorOpExists, "kubernetes.io/hostname"),
				},
			}))
		})
	})
}

func selectSchedulableNode(t *testing.T, ctx context.Context) (*corev1.Node, error) {
	nodes, err := TestClient(t).CoreV1().Nodes().List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		if node.Spec.Taints == nil {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("no node available")
}

func nodeSelector(key string, operator corev1.NodeSelectorOperator, value string) *corev1.NodeSelector {
	return &corev1.NodeSelector{
		NodeSelectorTerms: []corev1.NodeSelectorTerm{
			{
				MatchExpressions: []corev1.NodeSelectorRequirement{
					{
						Key:      key,
						Operator: operator,
						Values:   []string{value},
					},
				},
			},
		},
	}
}

func podAffinityTerm(key string, operator metav1.LabelSelectorOperator, topologyKey string) corev1.PodAffinityTerm {
	return corev1.PodAffinityTerm{
		LabelSelector: &metav1.LabelSelector{
			MatchExpressions: []metav1.LabelSelectorRequirement{
				{
					Key:      key,
					Operator: operator,
				},
			},
		},
		TopologyKey: topologyKey,
	}
}
