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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	messaging "knative.dev/eventing/pkg/apis/messaging/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

// Test that a Pipe can be changed and the changes are propagated to the Integration
func TestKameletChange(t *testing.T) {
	RegisterTestingT(t)

	knChannel := "test-kamelet-messages"
	timerSource := "my-timer-source"
	Expect(CreateTimerKamelet(ns, timerSource)()).To(Succeed())
	Expect(CreateKnativeChannel(ns, knChannel)()).To(Succeed())
	Expect(KamelRunWithID(operatorID, ns, "files/test-kamelet-display.groovy", "-w").Execute()).To(Succeed())

	from := corev1.ObjectReference{
		Kind:       "Kamelet",
		APIVersion: v1.SchemeGroupVersion.String(),
		Name:       timerSource,
	}

	to := corev1.ObjectReference{
		Kind:       "InMemoryChannel",
		Name:       knChannel,
		APIVersion: messaging.SchemeGroupVersion.String(),
	}

	timerPipe := "timer-binding"
	annotations := map[string]string{
		"trait.camel.apache.org/health.enabled":                 "true",
		"trait.camel.apache.org/health.readiness-initial-delay": "10",
	}

	// Create the Pipe
	Expect(BindKameletTo(ns, timerPipe, annotations, from, to, map[string]string{"message": "message is Hello"}, map[string]string{})()).To(Succeed())

	Eventually(IntegrationPodPhase(ns, timerPipe), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, "timer-binding", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationLogs(ns, "test-kamelet-display"), TestTimeoutShort).Should(ContainSubstring("message is Hello"))

	Eventually(PipeCondition(ns, timerPipe, v1.PipeConditionReady), TestTimeoutMedium).Should(And(
		WithTransform(PipeConditionStatusExtract, Equal(corev1.ConditionTrue)),
		WithTransform(PipeConditionReason, Equal(v1.IntegrationConditionDeploymentReadyReason)),
		WithTransform(PipeConditionMessage, Equal(fmt.Sprintf("1/1 ready replicas"))),
	))

	// Update the Pipe
	Expect(BindKameletTo(ns, "timer-binding", annotations, from, to, map[string]string{"message": "message is Hi"}, map[string]string{})()).To(Succeed())

	Eventually(IntegrationPodPhase(ns, "timer-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
	Eventually(IntegrationConditionStatus(ns, "timer-binding", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
	Eventually(IntegrationLogs(ns, "test-kamelet-display"), TestTimeoutShort).Should(ContainSubstring("message is Hi"))

	Eventually(PipeCondition(ns, timerPipe, v1.PipeConditionReady), TestTimeoutMedium).
		Should(And(
			WithTransform(PipeConditionStatusExtract, Equal(corev1.ConditionTrue)),
			WithTransform(PipeConditionReason, Equal(v1.IntegrationConditionDeploymentReadyReason)),
			WithTransform(PipeConditionMessage, Equal("1/1 ready replicas")),
		))

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
