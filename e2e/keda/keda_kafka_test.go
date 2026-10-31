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

func TestKafkaKedaAutoscale(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/kafka-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment", "my-cluster-kafka", "-n", ns, "--timeout=180s"))
		ExpectExecSucceed(t, g, Kubectl("exec", "deployment/my-cluster-kafka", "-n", ns, "--",
			"/opt/kafka/bin/kafka-topics.sh", "--create", "--if-not-exists",
			"--topic", "my-topic", "--bootstrap-server", "localhost:9092",
			"--partitions", "1", "--replication-factor", "1"))

		t.Run("Serverless Kafka Autoscale", func(t *testing.T) {
			consumerName := "keda-kafka-auto-discovery"
			producerName := "timer-to-kafka"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-kafkatopic-to-log-auto.yaml", "-n", ns))

			// Start consumer
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(PointTo(BeNumerically("==", 1)))

			// Scale down to 0 when idle
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))

			// Start producer to generate traffic
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-kafka.yaml", "-n", ns))
			g.Eventually(PipeConditionStatus(t, ctx, ns, producerName, v1.PipeConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Consumer scales up to 1
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(PointTo(BeNumerically("==", 1)))

			// Verify messages are being received and processed
			g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutShort).
				Should(ContainSubstring("Body is null"))

			// Stop producer
			ExpectExecSucceed(t, g, Kubectl("delete", "pipe", producerName, "-n", ns))

			// Scale back to 0
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))

			ExpectExecSucceed(t, g, Kubectl("delete", "pipe", "--all", "-n", ns))
			ExpectExecSucceed(t, g, Kubectl("delete", "it", "--all", "-n", ns))
		})

		ExpectExecSucceed(t, g, Kubectl("delete", "-f", "files/kafka-ephemeral.yaml", "-n", ns))
	}, "keda")
}

func TestKafkaKedaAutoDiscovery(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery Kafka", func(t *testing.T) {
			integrationName := "keda-kafka-auto-discovery"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-kafkatopic-to-log-auto.yaml", "-n", ns))

			// Wait for ScaledObject
			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			// Verify auto-discovered trigger
			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("kafka"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["topic"]).To(Equal("my-topic"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["bootstrapServers"]).To(Equal("my-cluster-kafka-bootstrap.keda.svc:9092"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["consumerGroup"]).To(Equal("auto-group"))

			ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		})
	}, "keda")
}

func TestKafkaKedaAutoDiscoveryWithManualTrigger(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery with manual trigger", func(t *testing.T) {
			integrationName := "keda-kafka-auto-discovery-with-manual"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-kafka-auto-discovery-with-manual.yaml", "-n", ns))

			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(2))

			var kafkaTrigger, cronTrigger interface{}
			for _, trigger := range scaledObj.Spec.Triggers {
				if trigger.Type == "kafka" {
					kafkaTrigger = trigger
				}
				if trigger.Type == "cron" {
					cronTrigger = trigger
				}
			}
			g.Expect(kafkaTrigger).NotTo(BeNil(), "kafka trigger should be auto-discovered")
			g.Expect(cronTrigger).NotTo(BeNil(), "cron trigger should exist from manual config")

			ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		})
	}, "keda")
}

func TestKafkaKedaAutoMetadata(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery with autoMetadata merge", func(t *testing.T) {
			integrationName := "keda-kafka-auto-metadata"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-kafka-auto-metadata.yaml", "-n", ns))

			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("kafka"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["topic"]).To(Equal("my-topic"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["bootstrapServers"]).To(Equal("my-cluster-kafka-bootstrap.keda.svc:9092"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["consumerGroup"]).To(Equal("metadata-group"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["lagThreshold"]).To(Equal("100"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["activationLagThreshold"]).To(Equal("10"))

			ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		})
	}, "keda")
}