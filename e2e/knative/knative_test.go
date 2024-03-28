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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestKnative(t *testing.T) {
	ctx := TestContext()
	g := NewWithT(t)

	knChannelMessages := "messages"
	knChannelWords := "words"
	g.Expect(CreateKnativeChannel(t, ctx, ns, knChannelMessages)()).To(Succeed())
	g.Expect(CreateKnativeChannel(t, ctx, ns, knChannelWords)()).To(Succeed())

	t.Run("Service combo", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knative2.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knative2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "knative2", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knative3.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knative3"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "knative3", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knative1.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knative1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "knative1", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
		// Correct logs
		g.Eventually(IntegrationLogs(t, ctx, ns, "knative1"), TestTimeoutMedium).Should(ContainSubstring("Received from 2: Hello from knative2"))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knative1"), TestTimeoutMedium).Should(ContainSubstring("Received from 3: Hello from knative3"))
		// Incorrect logs
		g.Consistently(IntegrationLogs(t, ctx, ns, "knative1"), 10*time.Second).ShouldNot(ContainSubstring("Received from 2: Hello from knative3"))
		g.Consistently(IntegrationLogs(t, ctx, ns, "knative1"), 10*time.Second).ShouldNot(ContainSubstring("Received from 3: Hello from knative2"))
		// Clean up
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Channel combo v1beta1", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knativech2.groovy").Execute()).To(Succeed())
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knativech1.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativech2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativech1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativech2"), TestTimeoutMedium).Should(ContainSubstring("Received: Hello from knativech1"))
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Channel combo get to post", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knativegetpost2.groovy").Execute()).To(Succeed())
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knativegetpost1.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativegetpost2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativegetpost1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativegetpost2"), TestTimeoutMedium).Should(ContainSubstring(`Received ""`))
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Multi channel chain", func(t *testing.T) {
		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/knativemultihop3.groovy").Execute()).To(Succeed())
		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/knativemultihop2.groovy").Execute()).To(Succeed())
		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/knativemultihop1.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativemultihop3"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativemultihop2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativemultihop1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativemultihop3"), TestTimeoutMedium).Should(ContainSubstring(`From messages: message`))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativemultihop3"), TestTimeoutMedium).Should(ContainSubstring(`From words: word`))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativemultihop3"), TestTimeoutMedium).Should(ContainSubstring(`From words: transformed message`))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativemultihop3"), 10*time.Second).ShouldNot(ContainSubstring(`From messages: word`))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativemultihop3"), 10*time.Second).ShouldNot(ContainSubstring(`From words: message`))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativemultihop3"), 10*time.Second).ShouldNot(ContainSubstring(`From messages: transformed message`))
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Flow", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/flow.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "flow"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "flow", camelv1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(v1.ConditionTrue))

		t.Run("Scale to zero", func(t *testing.T) {
			g.Eventually(IntegrationPod(t, ctx, ns, "flow"), TestTimeoutLong).Should(BeNil())
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Knative-service disabled", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/http_out.groovy", "-t", "knative-service.enabled=false").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "http-out"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(Service(t, ctx, ns, "http-out"), TestTimeoutShort).ShouldNot(BeNil())
		g.Consistently(KnativeService(t, ctx, ns, "http-out"), TestTimeoutShort).Should(BeNil())
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Knative-service priority", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/http_out.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "http-out"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(KnativeService(t, ctx, ns, "http-out"), TestTimeoutShort).ShouldNot(BeNil())
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Knative-service annotation", func(t *testing.T) {
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knative2.groovy", "-t", "knative-service.annotations.'haproxy.router.openshift.io/balance'=roundrobin").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knative2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(KnativeService(t, ctx, ns, "knative2"), TestTimeoutShort).ShouldNot(BeNil())
		ks := KnativeService(t, ctx, ns, "knative2")()
		annotations := ks.ObjectMeta.Annotations
		g.Expect(annotations["haproxy.router.openshift.io/balance"]).To(Equal("roundrobin"))
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunBroker(t *testing.T) {
	WithNewTestNamespaceWithKnativeBroker(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--trait-profile", "knative")).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(camelv1.IntegrationPlatformPhaseReady))

		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knativeevt1.groovy").Execute()).To(Succeed())
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/knativeevt2.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativeevt1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativeevt2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativeevt2"), TestTimeoutMedium).Should(ContainSubstring("Received 1: Hello 1"))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativeevt2"), TestTimeoutMedium).Should(ContainSubstring("Received 2: Hello 2"))
		g.Eventually(IntegrationLogs(t, ctx, ns, "knativeevt2")).ShouldNot(ContainSubstring("Received 1: Hello 2"))
		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
