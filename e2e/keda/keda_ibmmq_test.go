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

func TestIBMMQKedaAutoDiscoveryAndAutoscale(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		consumerName := "keda-it-ibmmq-to-log-auto"
		producerName := "timer-to-ibmmq"

		// Remove all test resources even if an assertion fails part-way through.
		t.Cleanup(func() {
			_ = Kubectl("delete", "it", producerName, "-n", ns, "--ignore-not-found=true").Run()
			_ = Kubectl("delete", "it", consumerName, "-n", ns, "--ignore-not-found=true").Run()
			_ = Kubectl("delete", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns, "--ignore-not-found=true").Run()
		})

		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment/ibmmq", "-n", ns, "--timeout=300s"))

		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-ibmmq-to-log-auto.yaml", "-n", ns))

		// Wait for Camel K to create the IBM MQ consumer and the auto-discovered ScaledObject.
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(ScaledObject(t, ctx, ns, consumerName), TestTimeoutLong).
			ShouldNot(BeNil())
		ExpectExecSucceed(t, g, Kubectl("wait", "--for=condition=Ready", "scaledobject/"+consumerName, "-n", ns, "--timeout=180s"))
		ExpectExecSucceed(t, g, Kubectl("wait", "--for=condition=ScalingActive", "hpa/keda-hpa-"+consumerName, "-n", ns, "--timeout=180s"))

		scaledObject := ScaledObject(t, ctx, ns, consumerName)()
		g.Expect(scaledObject).NotTo(BeNil())
		g.Expect(scaledObject.Spec.Triggers).To(HaveLen(1))
		g.Expect(scaledObject.Spec.Triggers[0].Type).To(Equal("ibmmq"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("queueName", "DEV.QUEUE.1"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("host", "https://ibmmq.keda.svc:9443/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("queueDepth", "1"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("unsafeSsl", "true"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).NotTo(HaveKey("username"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).NotTo(HaveKey("password"))

		// The queue is initially empty, so KEDA should scale the consumer to zero.
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 1)))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 0)))

		// Produce messages using the IBM MQ app user. KEDA should scale the
		// consumer back up when the queue depth becomes non-zero.
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-ibmmq.yaml", "-n", ns))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 1)))
		g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutShort).
			Should(ContainSubstring("IBM MQ scaler E2E message"))

		// Stop producing, let the consumer drain the queue, and verify scale-to-zero.
		ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 0)))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", consumerName, "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("delete", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns))
	}, "keda")
}
