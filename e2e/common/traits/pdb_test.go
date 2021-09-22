// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "knative"

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
	"net/http"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	policy "k8s.io/api/policy/v1beta1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestPodDisruptionBudgetTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"--name", name,
			"-t", "pdb.enabled=true",
			"-t", "pdb.min-available=2",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Check PodDisruptionBudget
		Eventually(podDisruptionBudget(ns, name), TestTimeoutShort).ShouldNot(BeNil())
		pdb := podDisruptionBudget(ns, name)()
		// Assert PDB Spec
		Expect(pdb.Spec.MinAvailable).To(PointTo(Equal(intstr.FromInt(2))))
		// Assert PDB Status
		Eventually(podDisruptionBudget(ns, name), TestTimeoutShort).
			Should(MatchFieldsP(IgnoreExtras, Fields{
				"Status": MatchFields(IgnoreExtras, Fields{
					"ObservedGeneration": BeNumerically("==", 1),
					"DisruptionsAllowed": BeNumerically("==", 0),
					"CurrentHealthy":     BeNumerically("==", 1),
					"DesiredHealthy":     BeNumerically("==", 2),
					"ExpectedPods":       BeNumerically("==", 1),
				}),
			}))

		// Scale Integration
		Expect(ScaleIntegration(ns, name, 2)).To(Succeed())
		Eventually(IntegrationPods(ns, name), TestTimeoutMedium).Should(HaveLen(2))
		Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 2)))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Check PodDisruptionBudget
		pdb = podDisruptionBudget(ns, name)()
		Expect(pdb).NotTo(BeNil())
		// Assert PDB Status according to the scale change
		Eventually(podDisruptionBudget(ns, name), TestTimeoutShort).
			Should(MatchFieldsP(IgnoreExtras, Fields{
				"Status": MatchFields(IgnoreExtras, Fields{
					"ObservedGeneration": BeNumerically("==", 1),
					"DisruptionsAllowed": BeNumerically("==", 0),
					"CurrentHealthy":     BeNumerically("==", 2),
					"DesiredHealthy":     BeNumerically("==", 2),
					"ExpectedPods":       BeNumerically("==", 2),
				}),
			}))

		// Eviction attempt
		pods := IntegrationPods(ns, name)()
		Expect(pods).To(HaveLen(2))
		err := TestClient().CoreV1().Pods(ns).Evict(TestContext, &policy.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name: pods[0].Name,
			},
		})
		Expect(err).To(MatchError(&errors.StatusError{
			ErrStatus: metav1.Status{
				Status:  "Failure",
				Message: "Cannot evict pod as it would violate the pod's disruption budget.",
				Reason:  "TooManyRequests",
				Code:    http.StatusTooManyRequests,
				Details: &metav1.StatusDetails{
					Causes: []metav1.StatusCause{
						{
							Type:    "DisruptionBudget",
							Message: "The disruption budget java needs 2 healthy pods and has 2 currently",
						},
					},
				},
			},
		}))

		// Scale Integration to Scale > PodDisruptionBudgetSpec.MinAvailable
		// for the eviction request to succeed once replicas are ready
		Expect(ScaleIntegration(ns, name, 3)).To(Succeed())
		Eventually(IntegrationPods(ns, name), TestTimeoutMedium).Should(HaveLen(3))
		Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 3)))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		pods = IntegrationPods(ns, name)()
		Expect(pods).To(HaveLen(3))
		Expect(TestClient().CoreV1().Pods(ns).Evict(TestContext, &policy.Eviction{
			ObjectMeta: metav1.ObjectMeta{
				Name: pods[0].Name,
			},
		})).To(Succeed())

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func podDisruptionBudget(ns string, name string) func() *policy.PodDisruptionBudget {
	return func() *policy.PodDisruptionBudget {
		pdb := policy.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: policy.SchemeGroupVersion.String(),
				Kind:       "PodDisruptionBudget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		err := TestClient().Get(TestContext, ctrl.ObjectKeyFromObject(&pdb), &pdb)
		if err != nil && errors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &pdb
	}
}
