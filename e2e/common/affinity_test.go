// +build integration

// To enable compilation of this file in Goland, go to "File -> Settings -> Go -> Build Tags & Vendoring -> Build Tags -> Custom tags" and add "integration"

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
	"fmt"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func TestAffinityTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())

		var hostname string
		if node, err := selectSchedulableNode(); err == nil {
			hostname = node.Labels["kubernetes.io/hostname"]
		} else {
			// if 'get nodes' is not allowed, just skip tests for node affinity
			hostname = ""
		}

		if hostname != "" {
			t.Run("Run Java with node affinity", func(t *testing.T) {
				RegisterTestingT(t)
				Expect(Kamel("run", "-n", ns, "files/Java.java",
					"--name", "java1",
					"-t", "affinity.enabled=true",
					"-t", fmt.Sprintf("affinity.node-affinity-labels=kubernetes.io/hostname in(%s)", hostname)).Execute()).Should(BeNil())
				Eventually(IntegrationPodPhase(ns, "java1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
				Eventually(IntegrationCondition(ns, "java1", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
				Eventually(IntegrationLogs(ns, "java1"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

				pod := IntegrationPod(ns, "java1")()
				Expect(pod.Spec.Affinity).ShouldNot(BeNil())
				Expect(pod.Spec.Affinity.NodeAffinity).Should(Equal(&v1.NodeAffinity{
					RequiredDuringSchedulingIgnoredDuringExecution: nodeSelector("kubernetes.io/hostname", v1.NodeSelectorOpIn, hostname),
				}))
				Expect(pod.Spec.NodeName).Should(Equal(hostname))

				Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
			})
		}

		t.Run("Run Java with pod affinity", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"--name", "java2",
				"-t", "affinity.enabled=true",
				"-t", "affinity.pod-affinity-labels=camel.apache.org/integration").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "java2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java2", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java2"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, "java2")()
			Expect(pod.Spec.Affinity).ShouldNot(BeNil())
			Expect(pod.Spec.Affinity.PodAffinity).Should(Equal(&v1.PodAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					podAffinityTerm("camel.apache.org/integration", metav1.LabelSelectorOpExists, "kubernetes.io/hostname"),
				},
			}))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("Run Java with pod anti affinity", func(t *testing.T) {
			RegisterTestingT(t)

			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"--name", "java3",
				"-t", "affinity.enabled=true",
				"-t", "affinity.pod-anti-affinity-labels=camel.apache.org/integration").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "java3"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java3", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java3"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, "java3")()
			Expect(pod.Spec.Affinity).ShouldNot(BeNil())
			Expect(pod.Spec.Affinity.PodAntiAffinity).Should(Equal(&v1.PodAntiAffinity{
				RequiredDuringSchedulingIgnoredDuringExecution: []v1.PodAffinityTerm{
					podAffinityTerm("camel.apache.org/integration", metav1.LabelSelectorOpExists, "kubernetes.io/hostname"),
				},
			}))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})
	})
}

func selectSchedulableNode() (*v1.Node, error) {
	nodes, err := TestClient().CoreV1().Nodes().List(TestContext, metav1.ListOptions{})
	if err != nil {
		return nil, err
	}
	for _, node := range nodes.Items {
		if node.Spec.Taints == nil {
			return &node, nil
		}
	}
	return nil, fmt.Errorf("No node available")
}

func nodeSelector(key string, operator v1.NodeSelectorOperator, value string) *v1.NodeSelector {
	return &v1.NodeSelector{
		NodeSelectorTerms: []v1.NodeSelectorTerm{
			{
				MatchExpressions: []v1.NodeSelectorRequirement{
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

func podAffinityTerm(key string, operator metav1.LabelSelectorOperator, topologyKey string) v1.PodAffinityTerm {
	return v1.PodAffinityTerm{
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
