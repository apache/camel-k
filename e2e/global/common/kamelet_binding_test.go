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

package common

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

func TestKameletBinding(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-kameletbinding"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		// Error Handler testing
		t.Run("test error handler", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(createErrorProducerKamelet(ns, "my-own-error-producer-source")()).To(Succeed())
			Expect(createLogKamelet(ns, "my-own-log-sink")()).To(Succeed())
			from := corev1.ObjectReference{
				Kind:       "Kamelet",
				Name:       "my-own-error-producer-source",
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			}

			to := corev1.ObjectReference{
				Kind:       "Kamelet",
				Name:       "my-own-log-sink",
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			}

			errorHandler := map[string]interface{}{
				"sink": map[string]interface{}{
					"endpoint": map[string]interface{}{
						"ref": map[string]string{
							"kind":       "Kamelet",
							"apiVersion": v1alpha1.SchemeGroupVersion.String(),
							"name":       "my-own-log-sink",
						},
						"properties": map[string]string{
							"loggerName": "kameletErrorHandler",
						},
					}}}

			t.Run("throw error test", func(t *testing.T) {
				RegisterTestingT(t)

				Expect(BindKameletToWithErrorHandler(ns, "throw-error-binding", map[string]string{},
					from, to,
					map[string]string{"message": "throw Error"}, map[string]string{"loggerName": "integrationLogger"},
					errorHandler)()).To(Succeed())

				Eventually(IntegrationPodPhase(ns, "throw-error-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns, "throw-error-binding"), TestTimeoutShort).Should(ContainSubstring("kameletErrorHandler"))
				Eventually(IntegrationLogs(ns, "throw-error-binding"), TestTimeoutShort).ShouldNot(ContainSubstring("integrationLogger"))

			})

			t.Run("don't throw error test", func(t *testing.T) {
				RegisterTestingT(t)

				Expect(BindKameletToWithErrorHandler(ns, "no-error-binding", map[string]string{},
					from, to,
					map[string]string{"message": "true"}, map[string]string{"loggerName": "integrationLogger"},
					errorHandler)()).To(Succeed())

				Eventually(IntegrationPodPhase(ns, "no-error-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				Eventually(IntegrationLogs(ns, "no-error-binding"), TestTimeoutShort).ShouldNot(ContainSubstring("kameletErrorHandler"))
				Eventually(IntegrationLogs(ns, "no-error-binding"), TestTimeoutShort).Should(ContainSubstring("integrationLogger"))

			})

			// Cleanup
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		// Kamelet binding with traits testing
		t.Run("test kamelet binding with trait", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(CreateTimerKamelet(ns, "my-own-timer-source")()).To(Succeed())
			// Log sink kamelet exists from previous test

			from := corev1.ObjectReference{
				Kind:       "Kamelet",
				Name:       "my-own-timer-source",
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			}

			to := corev1.ObjectReference{
				Kind:       "Kamelet",
				Name:       "my-own-log-sink",
				APIVersion: v1alpha1.SchemeGroupVersion.String(),
			}

			Expect(BindKameletTo(ns, "kb-with-traits",
				map[string]string{"trait.camel.apache.org/camel.properties": "[\"camel.prop1=a\",\"camel.prop2=b\"]"},
				from, to,
				map[string]string{"message": "hello from test"}, map[string]string{"loggerName": "integrationLogger"})()).
				To(Succeed())

			Eventually(IntegrationPodPhase(ns, "kb-with-traits"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "kb-with-traits"), TestTimeoutShort).Should(ContainSubstring("hello from test"))
			Eventually(IntegrationLogs(ns, "kb-with-traits"), TestTimeoutShort).Should(ContainSubstring("integrationLogger"))
		})
	})
}

func createLogKamelet(ns string, name string) func() error {
	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "kamelet:source",
			"steps": []map[string]interface{}{
				{
					"to": "log:{{loggerName}}",
				},
			},
		},
	}

	props := map[string]v1alpha1.JSONSchemaProp{
		"loggerName": {
			Type: "string",
		},
	}

	return CreateKamelet(ns, name, flow, props, nil)
}

func createErrorProducerKamelet(ns string, name string) func() error {
	props := map[string]v1alpha1.JSONSchemaProp{
		"message": {
			Type: "string",
		},
	}

	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "timer:tick",
			"steps": []map[string]interface{}{
				{
					"set-body": map[string]interface{}{
						"constant": "{{message}}",
					},
				},
				{
					"set-body": map[string]interface{}{
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
