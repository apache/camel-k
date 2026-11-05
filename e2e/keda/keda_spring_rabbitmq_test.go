//go:build integration
// +build integration

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

func TestSpringRabbitMQKedaAutoscale(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/rabbitmq-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment", "my-rabbitmq", "-n", ns, "--timeout=180s"))

		t.Run("Serverless Spring RabbitMQ Autoscale", func(t *testing.T) {
			consumerName := "keda-it-spring-rabbitmq-to-log-auto"
			producerName := "timer-to-spring-rabbitmq"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-spring-rabbitmq-to-log-auto.yaml", "-n", ns))

			// Trigger is auto-discovered from the consumer URI
			g.Eventually(ScaledObject(t, ctx, ns, consumerName), TestTimeoutMedium).
				ShouldNot(BeNil())
			scaledObj := ScaledObject(t, ctx, ns, consumerName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("rabbitmq"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["queueName"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["mode"]).To(Equal("QueueLength"))

			// Start consumer; it declares the exchange, queue and binding
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(PointTo(BeNumerically("==", 1)))

			// Scale down to 0 when idle
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))

			// Start producer to generate traffic
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-spring-rabbitmq.yaml", "-n", ns))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Consumer scales up to 1
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 1)))

			// Verify messages are being received and processed
			g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(ContainSubstring("Hello RabbitMQ"))

			// Stop producer
			ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))

			// Scale back to 0
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))

			ExpectExecSucceed(t, g, Kubectl("delete", "it", "--all", "-n", ns))
		})

		ExpectExecSucceed(t, g, Kubectl("delete", "-f", "files/rabbitmq-ephemeral.yaml", "-n", ns))
	}, "keda")
}
