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

package misc

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestPipe(t *testing.T) {
	RegisterTestingT(t)

	// Error Handler testing
	t.Run("test error handler", func(t *testing.T) {
		Expect(createErrorProducerKamelet(ns, "my-own-error-producer-source")()).To(Succeed())
		Expect(CreateLogKamelet(ns, "my-own-log-sink")()).To(Succeed())

		t.Run("throw error test", func(t *testing.T) {
			Expect(KamelBindWithID(operatorID, ns,
				"my-own-error-producer-source",
				"my-own-log-sink",
				"--error-handler", "sink:my-own-log-sink",
				"-p", "source.message=throw Error",
				"-p", "sink.loggerName=integrationLogger",
				"-p", "error-handler.loggerName=kameletErrorHandler",
				"--name", "throw-error-binding",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, "throw-error-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "throw-error-binding"), TestTimeoutShort).Should(ContainSubstring("kameletErrorHandler"))
			Eventually(IntegrationLogs(ns, "throw-error-binding"), TestTimeoutShort).ShouldNot(ContainSubstring("integrationLogger"))

		})

		t.Run("don't throw error test", func(t *testing.T) {
			Expect(KamelBindWithID(operatorID, ns,
				"my-own-error-producer-source",
				"my-own-log-sink",
				"--error-handler", "sink:my-own-log-sink",
				"-p", "source.message=true",
				"-p", "sink.loggerName=integrationLogger",
				"-p", "error-handler.loggerName=kameletErrorHandler",
				"--name", "no-error-binding",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, "no-error-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "no-error-binding"), TestTimeoutShort).ShouldNot(ContainSubstring("kameletErrorHandler"))
			Eventually(IntegrationLogs(ns, "no-error-binding"), TestTimeoutShort).Should(ContainSubstring("integrationLogger"))

		})
	})

	//Pipe with traits testing
	t.Run("test Pipe with trait", func(t *testing.T) {
		Expect(CreateTimerKamelet(ns, "my-own-timer-source")()).To(Succeed())
		// Log sink kamelet exists from previous test

		Expect(KamelBindWithID(operatorID, ns,
			"my-own-timer-source",
			"my-own-log-sink",
			"-p", "source.message=hello from test",
			"-p", "sink.loggerName=integrationLogger",
			"--annotation", "trait.camel.apache.org/camel.properties=[\"camel.prop1=a\",\"camel.prop2=b\"]",
			"--name", "kb-with-traits",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, "kb-with-traits"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, "kb-with-traits"), TestTimeoutShort).Should(ContainSubstring("hello from test"))
		Eventually(IntegrationLogs(ns, "kb-with-traits"), TestTimeoutShort).Should(ContainSubstring("integrationLogger"))
	})

	// Pipe with wrong spec
	t.Run("test Pipe with wrong spec", func(t *testing.T) {
		name := "bad-klb"
		kb := v1.NewPipe(ns, name)
		kb.Spec = v1.PipeSpec{}
		_, err := kubernetes.ReplaceResource(TestContext, TestClient(), &kb)
		Eventually(err).Should(BeNil())
		Eventually(PipePhase(ns, name), TestTimeoutShort).Should(Equal(v1.PipePhaseError))
		Eventually(PipeConditionStatus(ns, name, v1.PipeConditionReady), TestTimeoutShort).ShouldNot(Equal(corev1.ConditionTrue))
		Eventually(PipeCondition(ns, name, v1.PipeIntegrationConditionError), TestTimeoutShort).Should(
			WithTransform(PipeConditionMessage, And(
				ContainSubstring("could not determine source URI"),
				ContainSubstring("no ref or URI specified in endpoint"),
			)))
	})

	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}

func createErrorProducerKamelet(ns string, name string) func() error {
	props := map[string]v1.JSONSchemaProp{
		"message": {
			Type: "string",
		},
	}

	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "timer:tick",
			"steps": []map[string]interface{}{
				{
					"setBody": map[string]interface{}{
						"constant": "{{message}}",
					},
				},
				{
					"setBody": map[string]interface{}{
						"simple": "${mandatoryBodyAs(Boolean)}",
					},
				},
				{
					"to": "kamelet:sink",
				},
			},
		},
	}

	return CreateKamelet(ns, name, flow, props, nil)
}
