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
)

func TestServiceTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ns string) {

		t.Run("NodePort service", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true",
				"-t", "service.node-port=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(t, ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())

			Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Default service (ClusterIP)", func(t *testing.T) {
			// Service trait is enabled by default
			Expect(KamelRunWithID(t, operatorID, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(t, ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

			Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("NodePort service from Type", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true",
				"-t", "service.type=NodePort").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(t, ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())

			Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("ClusterIP service from Type", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true",
				"-t", "service.type=ClusterIP").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(t, ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

			// check integration schema does not contains unwanted default trait value.
			Eventually(UnstructuredIntegration(t, ns, "platform-http-server")).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ns, "platform-http-server")()
			serviceTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "service")
			Expect(serviceTrait).ToNot(BeNil())
			Expect(len(serviceTrait)).To(Equal(2))
			Expect(serviceTrait["enabled"]).To(Equal(true))
			Expect(serviceTrait["type"]).To(Equal("ClusterIP"))

			Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("LoadBalancer service from Type", func(t *testing.T) {
			Expect(KamelRunWithID(t, operatorID, ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true",
				"-t", "service.type=LoadBalancer").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(t, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(t, ns, corev1.ServiceTypeLoadBalancer), TestTimeoutLong).ShouldNot(BeEmpty())

			Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
