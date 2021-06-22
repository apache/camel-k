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
	"testing"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	messaging "knative.dev/eventing/pkg/apis/messaging/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

// Test that kamelet binding can be changed and changes propagated to integrations
func TestKameletChange(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(CreateTimerKamelet(ns, "timer-source")()).To(Succeed())
		Expect(CreateKnativeChannel(ns, "messages")()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/display.groovy", "-w").Execute()).To(Succeed())
		ref := v1.ObjectReference{
			Kind:       "InMemoryChannel",
			Name:       "messages",
			APIVersion: messaging.SchemeGroupVersion.String(),
		}
		Expect(BindKameletTo(ns, "timer-binding", "timer-source", ref, map[string]string{"message": "message is Hello"})()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-binding"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "timer-binding", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "display"), TestTimeoutShort).Should(ContainSubstring("message is Hello"))

		Expect(BindKameletTo(ns, "timer-binding", "timer-source", ref, map[string]string{"message": "message is Hi"})()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-binding"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "timer-binding", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "display"), TestTimeoutShort).Should(ContainSubstring("message is Hi"))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
