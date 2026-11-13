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

package keda

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestNatsKedaAutoscale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/nats-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment", "nats", "-n", ns, "--timeout=300s"))

		t.Run("NATS JetStream auto-discovered trigger", func(t *testing.T) {
			consumerName := "keda-it-nats-to-log-auto"
			producerName := "timer-to-nats"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", cloneAndReplaceNamespace(t, "files/keda-it-nats-to-log-auto.yaml", ns), "-n", ns))

			// The ScaledObject uses the trigger discovered from the NATS endpoint URI
			g.Eventually(ScaledObject(t, ctx, ns, consumerName), TestTimeoutMedium).ShouldNot(BeNil())
			scaledObj := ScaledObject(t, ctx, ns, consumerName)()
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("nats-jetstream"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["account"]).To(Equal("$G"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["stream"]).To(Equal("ORDERS"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["consumer"]).To(Equal("keda-durable"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["natsServerMonitoringEndpoint"]).To(Equal("nats." + ns + ".svc:8222"))

			// Start consumer; it creates the stream and the durable consumer
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Scale down to 0 when idle
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))

			// Start producer to build a backlog on the durable consumer
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", cloneAndReplaceNamespace(t, "files/timer-to-nats.yaml", ns), "-n", ns))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Consumer scales up to 1 and processes the messages
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 1)))
			g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(ContainSubstring("Hello NATS"))

			// Stop producer and scale back to 0
			ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))
		})
	})
}
