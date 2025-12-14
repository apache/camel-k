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

package kafka

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKafkaKedaAutoscale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// NOTE: all resources are local to kafka namespace

		// The scenario is the following:
		// 1. Start a Kafka consumer and wait to autoscale to 0 as no traffic
		// 2. Start a Kafka producer and verify the consumer scales up and consume accordingly
		// 3. Stop the Kafka producer and verify the consumer scales to 0 accordingly
		t.Run("Serverless Kafka", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-kafkatopic-to-log.yaml"))
			ns := "kafka"
			consumerName := "keda-kafkatopic-to-log"
			// Start consumer
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// It should scale to 0 after some time as there is no traffic
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 0)))
			// Start producer
			producerName := "timer-to-kafkatopic"
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-kafkatopic.yaml"))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			// The consumer will scale up
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Verify we are consuming some record (the body is null as the timer is pushing nothing)
			g.Eventually(IntegrationLogs(t, ctx, ns, consumerName)).Should(ContainSubstring("Body is null"))
			// Stop the producer
			g.Expect(Kamel(t, ctx, "delete", producerName, "-n", ns).Execute()).To(Succeed())
			// Consumer should scale back to 0 after some time as there is no longer traffic
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(gstruct.PointTo(BeNumerically("==", 0)))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}

func TestKafkaKedaAutoDiscovery(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery Kafka", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-kafka-auto-discovery.yaml"))
			ns := "kafka"
			integrationName := "keda-kafka-auto-discovery"

			// Wait for ScaledObject
			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			// Verify the auto-discovered
			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("kafka"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["topic"]).To(Equal("my-topic"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["bootstrapServers"]).To(Equal("my-cluster-kafka-bootstrap.kafka.svc:9092"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["consumerGroup"]).To(Equal("auto-group"))

			g.Expect(Kamel(t, ctx, "delete", integrationName, "-n", ns).Execute()).To(Succeed())
		})
	})
}
