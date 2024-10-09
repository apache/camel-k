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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	corev1 "k8s.io/api/core/v1"
)

func TestGarbageCollectorTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Delete outdated resources", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeClusterIP), TestTimeoutShort).ShouldNot(BeEmpty())
			g.Eventually(Deployment(t, ctx, ns, "platform-http-server"), TestTimeoutShort).ShouldNot(BeNil())
			genOneDeploymentUID := DeploymentUID(t, ctx, ns, "platform-http-server")()

			// Update integration and disable service trait - existing service should be garbage collected
			g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java", "-t", "service.enabled=false").Execute()).To(Succeed())
			g.Eventually(ServicesByType(t, ctx, ns, corev1.ServiceTypeClusterIP), TestTimeoutShort).Should(BeEmpty())
			g.Eventually(Deployment(t, ctx, ns, "platform-http-server"), TestTimeoutShort).ShouldNot(BeNil())

			// IMPORTANT: The Deployment UID must not change, otherwise we won't honor rolling upgrades as we delete and create a
			// new Deployment for every Integration change.
			g.Consistently(DeploymentUID(t, ctx, ns, "platform-http-server"), 3*time.Second, 1*time.Minute).
				Should(Equal(genOneDeploymentUID))
		})
	})
}
