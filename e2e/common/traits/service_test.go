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
	"context"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestServiceTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("NodePort service", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true",
				"-t", "service.node-port=true").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())
		})

		t.Run("Default service (ClusterIP)", func(t *testing.T) {
			// Service trait is enabled by default
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())
		})

		t.Run("NodePort service from Type", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true",
				"-t", "service.type=NodePort").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())
		})

		t.Run("ClusterIP service from Type", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true", "-t", "service.type=ClusterIP").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())
		})

		t.Run("LoadBalancer service from Type", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=true",
				"-t", "service.type=LoadBalancer").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeLoadBalancer), TestTimeoutLong).ShouldNot(BeEmpty())
		})
	})
}

func TestPortsServiceTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Service on port 8085", func(t *testing.T) {
			name := RandomizedSuffixName("svc")
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
				"-p", "quarkus.http.port=8085",
				"-t", "container.ports=hello;8085",
				"-t", "service.ports=hello;85;8085",
				"--name", name,
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)).
				Should(Equal(corev1.ConditionTrue))
			// We cannot use the health trait to make sure the application is ready to
			// get requests as we're sharing the service port.
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutMedium).Should(ContainSubstring("Listening on: http://0.0.0.0:8085"))

			response, err := TestClient(t).CoreV1().RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/services/%s:%d/proxy/hello/", ns, name, 85)).
				SetHeader("name", "service-test").
				Timeout(30 * time.Second).
				DoRaw(ctx)

			g.Expect(err).ToNot(HaveOccurred())
			g.Expect(string(response)).To(Equal("Hello service-test"))
		})

	})
}
