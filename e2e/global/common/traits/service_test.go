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

	. "github.com/apache/camel-k/e2e/support"
)

func TestServiceTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-service"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("NodePort service", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true",
				"-t", "service.node-port=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(ns, corev1.ServiceTypeNodePort), TestTimeoutLong).ShouldNot(BeEmpty())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Default service (ClusterIP)", func(t *testing.T) {
			// Service trait is enabled by default
			Expect(KamelRunWithID(operatorID, ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "platform-http-server"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			//
			// Service names can vary with the ExternalName Service
			// sometimes being created first and being given the root name
			//
			Eventually(ServicesByType(ns, corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
