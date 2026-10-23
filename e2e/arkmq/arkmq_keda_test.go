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

package arkmq

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestArkMQKedaAutoDiscovery(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery ArkMQ queue", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-arkmq-to-log-auto.yaml"))
			integrationName := "keda-arkmq-to-log-auto"

			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("artemis-queue"))
			// Auto-discovered from the amqp:queue URI
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["queueName"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerAddress"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["managementEndpoint"]).To(Equal("my-broker-hdls-svc.arkmq.svc:8161"))
			// Merged from autoMetadata
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerName"]).To(Equal("amq-broker"))

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationName, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 0)))
			producerName := "timer-to-arkmq"
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-arkmq.yaml"))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(IntegrationLogs(t, ctx, ns, integrationName)).Should(ContainSubstring("Body is null"))
			ExpectExecSucceed(t, g, Kubectl("delete", "pipe", producerName, "-n", ns))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutMedium).
				Should(gstruct.PointTo(BeNumerically("==", 0)))

			ExpectExecSucceed(t, g, Kubectl("delete", "pipe", integrationName, "-n", ns))
		})
	}, "arkmq")
}

func TestArkMQKedaAutoDiscoveryIntegration(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery ArkMQ queue from Integration", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-arkmq-to-log-auto.yaml"))
			integrationName := "keda-it-arkmq-to-log-auto"

			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("artemis-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["queueName"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerAddress"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["managementEndpoint"]).To(Equal("my-broker-hdls-svc.arkmq.svc:8161"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerName"]).To(Equal("amq-broker"))

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationName, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 0)))
			producerName := "timer-to-arkmq"
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-arkmq.yaml"))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(IntegrationLogs(t, ctx, ns, integrationName)).Should(ContainSubstring("Body is null"))
			ExpectExecSucceed(t, g, Kubectl("delete", "pipe", producerName, "-n", ns))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, integrationName), TestTimeoutMedium).
				Should(gstruct.PointTo(BeNumerically("==", 0)))

			ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		})
	}, "arkmq")
}
