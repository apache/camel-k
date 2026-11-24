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
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestAzureServiceBusQueueKedaAutoscale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		consumerName := "keda-it-azure-servicebus-queue-auto"
		producerName := "timer-to-azure-servicebus-queue"

		ExpectExecSucceed(t, g, Kubectl("apply", "-f",
			cloneAndReplaceNamespace(t, "files/keda-it-azure-servicebus-queue-auto.yaml", ns), "-n", ns))

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))

		// The generated trigger must contain names, not either connection string.
		g.Eventually(ScaledObject(t, ctx, ns, consumerName), TestTimeoutMedium).
			ShouldNot(BeNil())
		scaledObj := ScaledObject(t, ctx, ns, consumerName)()
		g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
		g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("azure-servicebus"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata["queueName"]).To(Equal("my-queue"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata["connectionFromEnv"]).To(Equal("SERVICEBUS_ADMIN_CONNECTION"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("connection"))
		for _, value := range scaledObj.Spec.Triggers[0].Metadata {
			g.Expect(value).NotTo(ContainSubstring("SAS_KEY_VALUE"))
		}

		// With no messages pending, KEDA should scale the consumer down to zero.
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 0)))

		// Send messages to the emulator. KEDA must query the queue depth and scale up.
		ExpectExecSucceed(t, g, Kubectl("apply", "-f",
			cloneAndReplaceNamespace(t, "files/timer-to-azure-servicebus-queue.yaml", ns), "-n", ns))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 1)))
		g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(ContainSubstring("camel-k-keda-azure-servicebus-queue-message"))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutLong).
			Should(PointTo(BeNumerically("==", 0)))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", consumerName, "-n", ns))
	})
}

func TestAzureServiceBusTopicKedaAutoscale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		consumerName := "keda-it-azure-servicebus-topic-auto"
		producerName := "timer-to-azure-servicebus-topic"

		ExpectExecSucceed(t, g, Kubectl("apply", "-f",
			cloneAndReplaceNamespace(t, "files/keda-it-azure-servicebus-topic-auto.yaml", ns), "-n", ns))

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))

		// Verify topic and subscription metadata are discovered, without exposing the connection.
		g.Eventually(ScaledObject(t, ctx, ns, consumerName), TestTimeoutMedium).
			ShouldNot(BeNil())
		scaledObj := ScaledObject(t, ctx, ns, consumerName)()
		g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
		g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("azure-servicebus"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata["topicName"]).To(Equal("my-topic"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata["subscriptionName"]).To(Equal("my-subscription"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata["connectionFromEnv"]).To(Equal("SERVICEBUS_ADMIN_CONNECTION"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("queueName"))
		g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("connection"))
		for _, value := range scaledObj.Spec.Triggers[0].Metadata {
			g.Expect(value).NotTo(ContainSubstring("SAS_KEY_VALUE"))
		}

		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 0)))

		// Publishing to the topic must activate the matching subscription scaler.
		ExpectExecSucceed(t, g, Kubectl("apply", "-f",
			cloneAndReplaceNamespace(t, "files/timer-to-azure-servicebus-topic.yaml", ns), "-n", ns))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 1)))
		g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(ContainSubstring("camel-k-keda-azure-servicebus-topic-message"))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutLong).
			Should(PointTo(BeNumerically("==", 0)))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", consumerName, "-n", ns))
	})
}
