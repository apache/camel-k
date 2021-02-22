// +build integration

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "knative"

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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestRunServiceCombo(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knative2.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knative2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "knative2", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Expect(Kamel("run", "-n", ns, "files/knative3.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knative3"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "knative3", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Expect(Kamel("run", "-n", ns, "files/knative1.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knative1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "knative1", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		// Correct logs
		Eventually(IntegrationLogs(ns, "knative1"), TestTimeoutMedium).Should(ContainSubstring("Received from 2: Hello from knative2"))
		Eventually(IntegrationLogs(ns, "knative1"), TestTimeoutMedium).Should(ContainSubstring("Received from 3: Hello from knative3"))
		// Incorrect logs
		Consistently(IntegrationLogs(ns, "knative1"), 10*time.Second).ShouldNot(ContainSubstring("Received from 2: Hello from knative3"))
		Consistently(IntegrationLogs(ns, "knative1"), 10*time.Second).ShouldNot(ContainSubstring("Received from 3: Hello from knative2"))
		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunChannelComboV1Alpha1(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(CreateKnativeChannelv1Alpha1(ns, "messages")()).To(Succeed())
		Expect(Kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativech2.groovy").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativech1.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knativech2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationPodPhase(ns, "knativech1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationLogs(ns, "knativech2"), TestTimeoutMedium).Should(ContainSubstring("Received: Hello from knativech1"))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunChannelComboV1Beta1(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(CreateKnativeChannelv1Beta1(ns, "messages")()).To(Succeed())
		Expect(Kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativech2.groovy").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativech1.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knativech2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationPodPhase(ns, "knativech1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationLogs(ns, "knativech2"), TestTimeoutMedium).Should(ContainSubstring("Received: Hello from knativech1"))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunChannelComboGetToPost(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(CreateKnativeChannelv1Beta1(ns, "messages")()).To(Succeed())
		Expect(Kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativegetpost2.groovy").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativegetpost1.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knativegetpost2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationPodPhase(ns, "knativegetpost1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationLogs(ns, "knativegetpost2"), TestTimeoutMedium).Should(ContainSubstring(`Received ""`))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

/*
// FIXME: uncomment when https://github.com/apache/camel-k-runtime/issues/69 is resolved
func TestRunMultiChannelChain(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(createKnativeChannel(ns, "messages")()).To(Succeed())
		Expect(createKnativeChannel(ns, "words")()).To(Succeed())
		Expect(kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(kamel("run", "-n", ns, "files/knativemultihop3.groovy").Execute()).To(Succeed())
		Expect(kamel("run", "-n", ns, "files/knativemultihop2.groovy").Execute()).To(Succeed())
		Expect(kamel("run", "-n", ns, "files/knativemultihop1.groovy").Execute()).To(Succeed())
		Eventually(integrationPodPhase(ns, "knativemultihop3"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(integrationPodPhase(ns, "knativemultihop2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(integrationPodPhase(ns, "knativemultihop1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(integrationLogs(ns, "knativemultihop3"), TestTimeoutMedium).Should(ContainSubstring(`From messages: message`))
		Eventually(integrationLogs(ns, "knativemultihop3"), TestTimeoutMedium).Should(ContainSubstring(`From words: word`))
		Eventually(integrationLogs(ns, "knativemultihop3"), TestTimeoutMedium).Should(ContainSubstring(`From words: transformed message`))
		Eventually(integrationLogs(ns, "knativemultihop3"), 10*time.Second).ShouldNot(ContainSubstring(`From messages: word`))
		Eventually(integrationLogs(ns, "knativemultihop3"), 10*time.Second).ShouldNot(ContainSubstring(`From words: message`))
		Eventually(integrationLogs(ns, "knativemultihop3"), 10*time.Second).ShouldNot(ContainSubstring(`From messages: transformed message`))
		Expect(kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
*/

func TestRunBroker(t *testing.T) {
	WithNewTestNamespaceWithKnativeBroker(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativeevt1.groovy").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/knativeevt2.groovy").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "knativeevt1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationPodPhase(ns, "knativeevt2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationLogs(ns, "knativeevt2"), TestTimeoutMedium).Should(ContainSubstring("Received 1: Hello 1"))
		Eventually(IntegrationLogs(ns, "knativeevt2"), TestTimeoutMedium).Should(ContainSubstring("Received 2: Hello 2"))
		Eventually(IntegrationLogs(ns, "knativeevt2")).ShouldNot(ContainSubstring("Received 1: Hello 2"))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunFlow(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/flow.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "flow"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "flow", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
