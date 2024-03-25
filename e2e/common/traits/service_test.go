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

func TestServiceTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-traits-service"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("NodePort service", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true", "-t", "service.node-port=true").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Default service (ClusterIP)", func(t *testing.T) {
			// Service trait is enabled by default
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("NodePort service from Type", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true", "-t", "service.type=NodePort").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("ClusterIP service from Type", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true", "-t", "service.type=ClusterIP").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, "platform-http-server")).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, "platform-http-server")()
			serviceTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "service")
			g.Expect(serviceTrait).ToNot(BeNil())
			g.Expect(len(serviceTrait)).To(Equal(2))
			g.Expect(serviceTrait["enabled"]).To(Equal(true))
			g.Expect(serviceTrait["type"]).To(Equal("ClusterIP"))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("LoadBalancer service from Type", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true", "-t", "service.type=LoadBalancer").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeLoadBalancer), TestTimeoutLong).ShouldNot(BeEmpty())

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
