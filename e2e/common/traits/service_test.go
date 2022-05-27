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
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("Default service (ClusterIP)", func(t *testing.T) {
			// Service trait is enabled by default
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "platform-http-server"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			service := Service(ns, "platform-http-server")
			Eventually(service, TestTimeoutShort).ShouldNot(BeNil())
			Expect(service().Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("NodePort service", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/PlatformHttpServer.java",
				"-t", "service.enabled=true",
				"-t", "service.node-port=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "platform-http-server"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(Service(ns, "platform-http-server"), TestTimeoutShort).ShouldNot(BeNil())
			Eventually(ServiceType(ns, "platform-http-server"), TestTimeoutShort).Should(Equal(corev1.ServiceTypeNodePort))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
