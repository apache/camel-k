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
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		consumerName := "keda-it-ibmmq-to-log-auto"
		producerName := "timer-to-ibmmq"

		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment/ibmmq", "-n", ns, "--timeout=300s"))

		ExpectExecSucceed(t, g, Kubectl(
			"apply", "-f",
			cloneAndReplaceNamespace(t, "files/keda-it-ibmmq-to-log-auto.yaml", ns),
			"-n", ns,
		))

		g.Eventually(
			IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady),
			TestTimeoutLong,
		).Should(Equal(corev1.ConditionTrue))

		// Verify autoscaling through observable Integration replica changes.
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 1)))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 0)))

		ExpectExecSucceed(t, g, Kubectl(
			"apply", "-f",
			cloneAndReplaceNamespace(t, "files/timer-to-ibmmq.yaml", ns),
			"-n", ns,
		))

		g.Eventually(
			IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady),
			TestTimeoutLong,
		).Should(Equal(corev1.ConditionTrue))

		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutShort).
			Should(PointTo(BeNumerically("==", 1)))
		g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutShort).
			Should(ContainSubstring("IBM MQ scaler E2E message"))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
			Should(PointTo(BeNumerically("==", 0)))
	})
}
