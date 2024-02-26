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
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestIstioTrait(t *testing.T) {
	RegisterTestingT(t)

	t.Run("Run Java with Istio", func(t *testing.T) {
		name := RandomizedSuffixName("java")
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "istio.enabled=true").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		pod := IntegrationPod(ns, name)()
		Expect(pod.ObjectMeta.Annotations).NotTo(BeNil())
		annotations := pod.ObjectMeta.Annotations
		Expect(annotations["sidecar.istio.io/inject"]).To(Equal("true"))
		Expect(annotations["traffic.sidecar.istio.io/includeOutboundIPRanges"]).To(Equal("10.0.0.0/8,172.16.0.0/12,192.168.0.0/16"))

		// check integration schema does not contains unwanted default trait value.
		Eventually(UnstructuredIntegration(ns, name)).ShouldNot(BeNil())
		unstructuredIntegration := UnstructuredIntegration(ns, name)()
		istioTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "istio")
		Expect(istioTrait).ToNot(BeNil())
		Expect(len(istioTrait)).To(Equal(1))
		Expect(istioTrait["enabled"]).To(Equal(true))

	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
