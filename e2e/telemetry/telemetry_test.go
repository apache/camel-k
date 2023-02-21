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

package telemetry

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
)

func TestTelemetryTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-telemetry"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		// Check service is available
		Eventually(ServicesByType("otlp", corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

		// Create integration and activate traces by telemetry trait

		Expect(KamelRunWithID(operatorID, ns, "files/rest-consumer.yaml",
			"--name", "rest-consumer",
			"-t", "telemetry.enabled=true",
			"-t", "telemetry.endpoint=http://opentelemetrycollector.otlp.svc.cluster.local:4317").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "rest-consumer"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

		name := "Bob"
		Expect(KamelRunWithID(operatorID, ns, "files/rest-producer.yaml",
			"-p", "serviceName=rest-consumer",
			"-p", "name="+name,
			"--name", "rest-producer").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "rest-producer"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, "rest-consumer"), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("get %s", name)))
		Eventually(IntegrationLogs(ns, "rest-producer"), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("%s Doe", name)))

		// Find opentelemetrycollector pod : the exporter is configured to log traces with detailed verborsity.
		pod, err := Pod("otlp", "opentelemetrycollector")()
		Expect(err).To(BeNil())
		Expect(pod).NotTo(BeNil())

		// Ensured logs in opentelemetrycollector pod are present
		Eventually(TailedLogs(pod.Namespace, pod.Name, 100), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("http.target: Str(/customers/%s)", name)))
		Eventually(TailedLogs(pod.Namespace, pod.Name, 100), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("http.url: Str(http://rest-consumer/customers/%s)", name)))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
