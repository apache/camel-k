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
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client/camel/clientset/versioned"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

func TestPipe(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Error Handler testing
		t.Run("test error handler", func(t *testing.T) {
			g.Expect(createErrorProducerKamelet(t, ctx, ns, "my-own-error-producer-source")()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, "my-own-log-sink")()).To(Succeed())

			t.Run("throw error test", func(t *testing.T) {
				g.Expect(KamelBind(t, ctx, ns, "my-own-error-producer-source", "my-own-log-sink",
					"--error-handler", "sink:my-own-log-sink",
					"-p", "source.message=throw Error",
					"-p", "sink.loggerName=integrationLogger",
					"-p", "error-handler.loggerName=kameletErrorHandler",
					// Needed in the test to make sure to do the right string comparison later
					"-t", "logging.color=false",
					"--name", "throw-error-binding").Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, "throw-error-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, "throw-error-binding"), TestTimeoutShort).Should(ContainSubstring("[kameletErrorHandler] (Camel (camel-1) thread #1 - timer://tick)"))
				g.Eventually(IntegrationLogs(t, ctx, ns, "throw-error-binding"), TestTimeoutShort).ShouldNot(ContainSubstring("[integrationLogger] (Camel (camel-1) thread #1 - timer://tick)"))

			})

			t.Run("don't throw error test", func(t *testing.T) {
				g.Expect(KamelBind(t, ctx, ns, "my-own-error-producer-source", "my-own-log-sink",
					"--error-handler", "sink:my-own-log-sink",
					"-p", "source.message=true",
					"-p", "sink.loggerName=integrationLogger",
					"-p", "error-handler.loggerName=kameletErrorHandler",
					// Needed in the test to make sure to do the right string comparison later
					"-t", "logging.color=false",
					"--name", "no-error-binding").Execute()).To(Succeed())

				g.Eventually(IntegrationPodPhase(t, ctx, ns, "no-error-binding"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
				g.Eventually(IntegrationLogs(t, ctx, ns, "no-error-binding"), TestTimeoutShort).ShouldNot(ContainSubstring("[kameletErrorHandler] (Camel (camel-1) thread #1 - timer://tick)"))
				g.Eventually(IntegrationLogs(t, ctx, ns, "no-error-binding"), TestTimeoutShort).Should(ContainSubstring("[integrationLogger] (Camel (camel-1) thread #1 - timer://tick)"))

			})
		})

		//Pipe with traits testing
		t.Run("test Pipe with trait", func(t *testing.T) {
			g.Expect(CreateTimerKamelet(t, ctx, ns, "my-own-timer-source")()).To(Succeed())
			// Log sink kamelet exists from previous test

			g.Expect(KamelBind(t, ctx, ns, "my-own-timer-source", "my-own-log-sink",
				"-p", "source.message=hello from test",
				"-p", "sink.loggerName=integrationLogger",
				"--trait", "camel.properties=[\"camel.prop1=a\",\"camel.prop2=b\"]",
				"--name", "kb-with-traits").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, "kb-with-traits"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "kb-with-traits"), TestTimeoutShort).Should(ContainSubstring("hello from test"))
			g.Eventually(IntegrationLogs(t, ctx, ns, "kb-with-traits"), TestTimeoutShort).Should(ContainSubstring("integrationLogger"))
		})

		// Pipe with wrong spec
		t.Run("test Pipe with wrong spec", func(t *testing.T) {
			name := RandomizedSuffixName("bad-klb")
			kb := v1.NewPipe(ns, name)
			kb.Spec = v1.PipeSpec{}
			_, err := kubernetes.ReplaceResource(ctx, TestClient(t), &kb)
			g.Eventually(err).Should(BeNil())
			g.Eventually(PipePhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.PipePhaseError))
			g.Eventually(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutShort).ShouldNot(Equal(corev1.ConditionTrue))
			g.Eventually(PipeCondition(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(
				WithTransform(PipeConditionMessage, ContainSubstring("no ref or URI specified in endpoint")))
		})

	})
}

func TestPipeWithImage(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		bindingID := "with-image-binding"

		t.Run("run with initial image", func(t *testing.T) {
			expectedImage := "quay.io/fuse_qe/echo-server:0.3.2"

			g.Expect(KamelBind(t, ctx, ns, "my-own-timer-source", "my-own-log-sink",
				"--trait", "container.image="+expectedImage, "--trait", "jvm.enabled=false",
				"--trait", "kamelets.enabled=false", "--trait", "dependencies.enabled=false",
				"--annotation", "test=1", "--name", bindingID).Execute()).To(Succeed())

			g.Eventually(IntegrationGeneration(t, ctx, ns, bindingID)).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(Integration(t, ctx, ns, bindingID)).Should(WithTransform(Annotations,
				HaveKeyWithValue("test", "1"),
			))
			g.Eventually(IntegrationStatusImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, bindingID), TestTimeoutShort).
				Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
		})

		t.Run("run with new image", func(t *testing.T) {
			expectedImage := "quay.io/fuse_qe/echo-server:0.3.3"

			g.Expect(KamelBind(t, ctx, ns, "my-own-timer-source", "my-own-log-sink",
				"--trait", "container.image="+expectedImage, "--trait", "jvm.enabled=false",
				"--trait", "kamelets.enabled=false", "--trait", "dependencies.enabled=false",
				"--annotation", "test=2", "--name", bindingID).Execute()).To(Succeed())
			g.Eventually(IntegrationGeneration(t, ctx, ns, bindingID)).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(Integration(t, ctx, ns, bindingID)).Should(WithTransform(Annotations,
				HaveKeyWithValue("test", "2"),
			))
			g.Eventually(IntegrationStatusImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, bindingID), TestTimeoutShort).
				Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
		})
	})
}

func TestPipeScale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("timer2log")
		g.Expect(KamelBind(t, ctx, ns, "timer-source?message=HelloPipe", "log-sink", "--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("HelloPipe"))

		t.Run("Update Pipe scale spec", func(t *testing.T) {
			g.Expect(ScalePipe(t, ctx, ns, name, 3)).To(Succeed())
			// Check the scale cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Check it also cascades into the Pipe scale subresource Status field
			g.Eventually(PipeStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Check the readiness condition becomes truthy back
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			// Finally check the readiness condition becomes truthy back onPipe
			g.Eventually(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		})

		t.Run("ScalePipe with polymorphic client", func(t *testing.T) {
			scaleClient, err := TestClient(t).ScalesClient()
			g.Expect(err).To(BeNil())

			// Patch the integration scale subresource
			patch := "{\"spec\":{\"replicas\":2}}"
			_, err = scaleClient.Scales(ns).Patch(ctx, v1.SchemeGroupVersion.WithResource("Pipes"), name, types.MergePatchType, []byte(patch), metav1.PatchOptions{})
			g.Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling
			g.Expect(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			g.Eventually(IntegrationSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Then check it cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(2))
			// Check it cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
			// Finally check it cascades into the Pipe scale subresource Status field
			g.Eventually(PipeStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 2)))
		})

		t.Run("ScalePipe with Camel K client", func(t *testing.T) {
			camel, err := versioned.NewForConfig(TestClient(t).GetConfig())
			g.Expect(err).To(BeNil())

			// Getter
			PipeScale, err := camel.CamelV1().Pipes(ns).GetScale(ctx, name, metav1.GetOptions{})
			g.Expect(err).To(BeNil())
			g.Expect(PipeScale.Spec.Replicas).To(BeNumerically("==", 2))
			g.Expect(PipeScale.Status.Replicas).To(BeNumerically("==", 2))

			// Setter
			PipeScale.Spec.Replicas = 1
			_, err = camel.CamelV1().Pipes(ns).UpdateScale(ctx, name, PipeScale, metav1.UpdateOptions{})
			g.Expect(err).To(BeNil())

			// Check the readiness condition is still truthy as down-scaling inPipe
			g.Expect(PipeConditionStatus(t, ctx, ns, name, v1.PipeConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Pipe scale subresource Spec field
			g.Eventually(PipeSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Check the readiness condition is still truthy as down-scaling
			g.Expect(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)()).To(Equal(corev1.ConditionTrue))
			// Check the Integration scale subresource Spec field
			g.Eventually(IntegrationSpecReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			// Then check it cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(1))
			// Finally check it cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
		})
	})
}

func createErrorProducerKamelet(t *testing.T, ctx context.Context, ns string, name string) func() error {
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

	return CreateKamelet(t, ctx, ns, name, flow, props, nil)
}
