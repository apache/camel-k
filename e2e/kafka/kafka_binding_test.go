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
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKafka(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// NOTE: all resources are local to kafka namespace
		t.Run("Strimzi Kafka resource", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-kafka.yaml"))
			// Wait for the readiness of the Integration
			g.Eventually(IntegrationConditionStatus(t, ctx, "kafka", "timer-to-kafka", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/kafka-to-log.yaml"))
			g.Eventually(IntegrationConditionStatus(t, ctx, "kafka", "kafka-to-log", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			// Verify we are consuming some record (the body is null as the timer is pushing nothing)
			g.Eventually(IntegrationLogs(t, ctx, "kafka", "kafka-to-log")).Should(ContainSubstring("Body is null"))

			g.Expect(Kamel(t, ctx, "delete", "kafka-to-log", "-n", "kafka").Execute()).To(Succeed())
		})

		t.Run("Strimzi KafkaTopic resource", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-kafkatopic.yaml"))
			// Wait for the readiness of the Integration
			g.Eventually(IntegrationConditionStatus(t, ctx, "kafka", "timer-to-kafkatopic", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/kafkatopic-to-log.yaml"))
			g.Eventually(IntegrationConditionStatus(t, ctx, "kafka", "kafkatopic-to-log", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			// Verify we are consuming some record (the body is null as the timer is pushing nothing)
			g.Eventually(IntegrationLogs(t, ctx, "kafka", "kafkatopic-to-log")).Should(ContainSubstring("Body is null"))

			g.Expect(Kamel(t, ctx, "delete", "kafkatopic-to-log", "-n", "kafka").Execute()).To(Succeed())
		})
	})
}
