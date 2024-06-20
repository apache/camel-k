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
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestTelemetryTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-trait-telemetry"
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		// Check service is available
		g.Eventually(ServicesByType(t, ctx, "otlp", corev1.ServiceTypeClusterIP), TestTimeoutLong).ShouldNot(BeEmpty())

		// Create integration and activate traces by telemetry trait

		g.Expect(KamelRunWithID(t, ctx, operatorID, ns,
			"files/rest-consumer.yaml", "--name", "rest-consumer",
			"-t", "telemetry.enabled=true",
			"-t", "telemetry.endpoint=http://opentelemetrycollector.otlp:4317").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "rest-consumer"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

		name := "Bob"
		serviceName := fmt.Sprintf("rest-consumer.%s", ns)
		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/rest-producer.yaml",
			"-p", fmt.Sprintf("serviceName=%s", serviceName),
			"-p", "name="+name,
			"--name", "rest-producer").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "rest-producer"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "rest-consumer"), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("get %s", name)))
		g.Eventually(IntegrationLogs(t, ctx, ns, "rest-producer"), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("%s Doe", name)))

		// Find opentelemetry collector pod : the exporter is configured to log traces with detailed verbosity.
		pod, err := Pod(t, ctx, "otlp", "opentelemetrycollector")()
		g.Expect(err).To(BeNil())
		g.Expect(pod).NotTo(BeNil())

		// Ensured logs in opentelemetry collector pod are present
		g.Eventually(TailedLogs(t, ctx, pod.Namespace, pod.Name, 100), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("http.target: Str(/customers/%s)", name)))
		g.Eventually(TailedLogs(t, ctx, pod.Namespace, pod.Name, 100), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("http.url: Str(http://%s/customers/%s)", serviceName, name)))
	})
}
