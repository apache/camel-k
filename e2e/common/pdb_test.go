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

package common

import (
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	"k8s.io/apimachinery/pkg/util/intstr"

	v1 "k8s.io/api/core/v1"
	"k8s.io/api/policy/v1beta1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestPodDisruptionBudget(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		Expect(Kamel("install", "-n", ns).Execute()).To(BeNil())
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"--name", name,
			"-t", "pdb.enabled=true",
			"-t", "pdb.min-available=2",
		).Execute()).To(BeNil())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Check PodDisruptionBudget
		Eventually(podDisruptionBudget(ns, name), TestTimeoutShort).ShouldNot(BeNil())
		pdb := podDisruptionBudget(ns, name)()
		// Assert PDB Spec
		Expect(pdb.Spec.MinAvailable).To(gstruct.PointTo(Equal(intstr.FromInt(2))))
		// Assert PDB Status
		Eventually(podDisruptionBudget(ns, name), TestTimeoutShort).Should(gstruct.PointTo(gstruct.MatchFields(
			gstruct.IgnoreExtras,
			gstruct.Fields{
				"Status": Equal(v1beta1.PodDisruptionBudgetStatus{
					ObservedGeneration: 1,
					DisruptionsAllowed: 0,
					CurrentHealthy:     1,
					DesiredHealthy:     2,
					ExpectedPods:       1,
				}),
			}),
		))

		// Scale Integration
		Expect(UpdateIntegration(ns, name, func(it *camelv1.Integration) {
			replicas := int32(2)
			it.Spec.Replicas = &replicas
		})).To(BeNil())
		Eventually(IntegrationPods(ns, name), TestTimeoutMedium).Should(HaveLen(2))
		Eventually(IntegrationStatusReplicas(ns, name), TestTimeoutShort).
			Should(gstruct.PointTo(BeNumerically("==", 2)))

		// Check PodDisruptionBudget
		pdb = podDisruptionBudget(ns, name)()
		Expect(pdb).NotTo(BeNil())
		// Assert PDB Status according to the scale change
		Eventually(podDisruptionBudget(ns, name), TestTimeoutShort).Should(gstruct.PointTo(gstruct.MatchFields(
			gstruct.IgnoreExtras,
			gstruct.Fields{
				"Status": Equal(v1beta1.PodDisruptionBudgetStatus{
					ObservedGeneration: 1,
					DisruptionsAllowed: 0,
					CurrentHealthy:     2,
					DesiredHealthy:     2,
					ExpectedPods:       2,
				}),
			}),
		))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(BeNil())
	})
}

func podDisruptionBudget(ns string, name string) func() *v1beta1.PodDisruptionBudget {
	return func() *v1beta1.PodDisruptionBudget {
		pdb := v1beta1.PodDisruptionBudget{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1beta1.SchemeGroupVersion.String(),
				Kind:       "PodDisruptionBudget",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ns,
				Name:      name,
			},
		}
		key, err := client.ObjectKeyFromObject(&pdb)
		if err != nil {
			panic(err)
		}
		err = TestClient().Get(TestContext, key, &pdb)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &pdb
	}
}
