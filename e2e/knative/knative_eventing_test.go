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

package knative

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestKnativeEventingChannel(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		const (
			channelName  = "camel-k-eventing"
			consumerName = "knative-channel-consumer"
			producerName = "knative-channel-producer"
			message      = "camel-k-knative-eventing-e2e"
		)

		// The Knative workflow installs the in-memory channel implementation.
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/knative-channel.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl(
			"wait", "--for=condition=Ready", "channel/"+channelName, "--timeout=120s", "-n", ns,
		))

		// Start the consumer first so it is subscribed before the producer sends its event.
		// Disable the Knative trait to avoid activating its deprecated behavior; the route
		// uses Camel's Knative component directly.
		g.Expect(KamelRun(t, ctx, ns, "files/knative-channel-consumer.yaml",
			"--name", consumerName,
			"-t", "knative.enabled=false",
		).Execute()).To(Succeed())
		g.Eventually(
			IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady),
			TestTimeoutMedium,
		).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, consumerName), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		// Ensure Knative has reconciled the subscription before publishing the event.
		ExpectExecSucceed(t, g, Kubectl(
			"wait", "--for=condition=Ready", "subscription/"+channelName+"-"+consumerName, "--timeout=120s", "-n", ns,
		))

		g.Expect(KamelRun(t, ctx, ns, "files/knative-channel-producer.yaml",
			"--name", producerName,
			"-t", "knative.enabled=false",
		).Execute()).To(Succeed())
		g.Eventually(
			IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady),
		).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, producerName), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))

		// Verify that the event sent to the channel reaches the consumer Integration.
		g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutLong).
			Should(ContainSubstring(message))
	})
}
