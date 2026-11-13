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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestActiveMQKedaAutoscale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/activemq-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment", "my-activemq", "-n", ns, "--timeout=180s"))

		t.Run("Serverless ActiveMQ Autoscale", func(t *testing.T) {
			consumerName := "keda-it-activemq-to-log-auto"
			producerName := "timer-to-activemq"

			// Start producer to generate traffic and populate the queue
			producerFile := cloneAndReplaceNamespace(t, "files/timer-to-activemq.yaml", ns)
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", producerFile, "-n", ns))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Start consumer; queue is non-empty so KEDA keeps replicas active
			consumerFile := cloneAndReplaceNamespace(t, "files/keda-it-activemq-to-log-auto.yaml", ns)
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", consumerFile, "-n", ns))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 1)))

			// Verify messages are being received and processed
			g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(ContainSubstring("Body is null"))

			// Stop producer to allow consumer to drain the queue
			ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))

			// Once idle, KEDA scales the consumer down to 0
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))
		})
	})
}

func TestActiveMQKedaAutoDiscovery(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery ActiveMQ", func(t *testing.T) {
			integrationName := "keda-it-activemq-to-log-auto"

			consumerFile := cloneAndReplaceNamespace(t, "files/keda-it-activemq-to-log-auto.yaml", ns)
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", consumerFile, "-n", ns))

			// Wait for ScaledObject
			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			// Verify auto-discovered trigger
			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("activemq"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["destinationName"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["managementEndpoint"]).To(Equal(fmt.Sprintf("my-activemq-hdls-svc.%s.svc:8161", ns)))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerName"]).To(Equal("localhost"))
		})
	})
}
