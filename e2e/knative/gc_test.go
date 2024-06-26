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

package knative

import (
	"context"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	. "github.com/onsi/gomega"
)

func TestGarbageCollectResources(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		integration := "platform-http-server"
		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "knative-service.enabled=false").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, integration), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integration, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		g.Eventually(KnativeService(t, ctx, ns, integration), TestTimeoutMedium).Should(BeNil())
		g.Eventually(ServiceType(t, ctx, ns, integration), TestTimeoutMedium).Should(Equal(corev1.ServiceTypeClusterIP))

		// Update integration and enable knative service trait - existing arbitrary service should be garbage collected
		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())

		g.Eventually(KnativeService(t, ctx, ns, integration), TestTimeoutShort).ShouldNot(BeNil())
		g.Eventually(ServiceType(t, ctx, ns, integration), TestTimeoutShort).Should(Equal(corev1.ServiceTypeExternalName))

		g.Eventually(IntegrationPodPhase(t, ctx, ns, integration), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integration, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Disable knative service trait again - this time knative service should be garbage collected
		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "knative-service.enabled=false").Execute()).To(Succeed())

		g.Eventually(KnativeService(t, ctx, ns, integration), TestTimeoutMedium).Should(BeNil())
		g.Eventually(ServiceType(t, ctx, ns, integration), TestTimeoutMedium).Should(Equal(corev1.ServiceTypeClusterIP))

		g.Eventually(IntegrationPodPhase(t, ctx, ns, integration), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integration, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	})
}
