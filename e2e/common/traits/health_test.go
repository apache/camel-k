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
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestHealthTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {

		t.Run("Readiness condition with stopped route scaled", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
				"-t", "jvm.agents=jolokia;https://repo1.maven.org/maven2/org/jolokia/jolokia-agent-jvm/2.3.0/jolokia-agent-jvm-2.3.0-javaagent.jar;host=*",
				"-t", "container.ports=jolokia;8778",
				"--name", name).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Expect(ScaleIntegration(t, ctx, ns, name, 3)).To(Succeed())
			// Check the readiness condition becomes falsy
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
			// Check the scale cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ctx, ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Finally check the readiness condition becomes truthy back
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))

			pods := IntegrationPods(t, ctx, ns, name)()

			t.Logf("Stopping routes for integration %s/%s (%d)", ns, name, len(pods))

			for _, pod := range pods {
				t.Logf("Stopping route on integration pod %s/%s", pod.Namespace, pod.Name)
				// Stop the Camel route
				request := map[string]string{
					"type":      "exec",
					"mbean":     "org.apache.camel:context=camel-1,name=\"route1\",type=routes",
					"operation": "stop()",
				}
				body, err := json.Marshal(request)
				g.Expect(err).To(BeNil())

				response, err := TestClient(t).CoreV1().RESTClient().Post().
					AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/jolokia/", ns, pod.Name)).
					Timeout(30 * time.Second).
					Body(body).
					DoRaw(ctx)
				g.Expect(err).To(BeNil())
				g.Expect(response).To(ContainSubstring(`"status":200`))

				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionFalse))

				t.Logf("Route on integration pod %s/%s stopped", pod.Namespace, pod.Name)
			}

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionFalse))
		})

		t.Run("Readiness condition with stopped route", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
				"-t", "jvm.agents=jolokia;https://repo1.maven.org/maven2/org/jolokia/jolokia-agent-jvm/2.3.0/jolokia-agent-jvm-2.3.0-javaagent.jar;host=*",
				"-t", "container.ports=jolokia;8778",
				"--name", name).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, name)()

			// Stop the Camel route
			request := map[string]string{
				"type":      "exec",
				"mbean":     "org.apache.camel:context=camel-1,name=\"route1\",type=routes",
				"operation": "stop()",
			}
			body, err := json.Marshal(request)
			g.Expect(err).To(BeNil())

			response, err := TestClient(t).CoreV1().RESTClient().Post().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/jolokia/", ns, pod.Name)).
				Timeout(30 * time.Second).
				Body(body).
				DoRaw(ctx)
			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned false
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
		})

		t.Run("Readiness condition with stopped pipe", func(t *testing.T) {
			name := RandomizedSuffixName("stopped-binding")
			source := RandomizedSuffixName("my-health-timer-source")
			sink := RandomizedSuffixName("my-health-log-sink")

			g.Expect(CreateTimerKamelet(t, ctx, ns, source)()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, sink)()).To(Succeed())

			g.Expect(KamelBind(t, ctx, ns, source, sink,
				"-p", "source.message=Magicstring!",
				"-p", "sink.loggerName=binding",
				"-t", "jvm.agents=jolokia;https://repo1.maven.org/maven2/org/jolokia/jolokia-agent-jvm/2.3.0/jolokia-agent-jvm-2.3.0-javaagent.jar;host=*",
				"-t", "container.ports=jolokia;8778",
				"--name", name).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, name)()

			// Stop the Camel route
			request := map[string]string{
				"type":      "exec",
				"mbean":     "org.apache.camel:context=camel-1,name=\"binding\",type=routes",
				"operation": "stop()",
			}
			body, err := json.Marshal(request)
			g.Expect(err).To(BeNil())

			response, err := TestClient(t).CoreV1().RESTClient().Post().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/jolokia/", ns, pod.Name)).
				Timeout(30 * time.Second).
				Body(body).
				DoRaw(ctx)

			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned false
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
		})

		t.Run("Readiness condition with never ready route", func(t *testing.T) {
			name := RandomizedSuffixName("never-ready")

			g.Expect(KamelRun(t, ctx, ns, "files/NeverReady.java", "--name", name,
				"-p", "camel.health.routesEnabled=false",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Consistently(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), 1*time.Minute).
				Should(Equal(corev1.ConditionFalse))
		})

		t.Run("Startup condition with never ready route", func(t *testing.T) {
			name := RandomizedSuffixName("startup-probe-never-ready-route")

			g.Expect(KamelRun(t, ctx, ns, "files/NeverReady.java", "--name", name,
				"-t", "health.startup-probe-enabled=true",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
			g.Consistently(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), 1*time.Minute).Should(Equal(corev1.ConditionFalse))
		})

		t.Run("Startup condition with ready route", func(t *testing.T) {
			name := RandomizedSuffixName("startup-probe-ready-route")

			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name,
				"-t", "health.startup-probe-enabled=true",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		})

	})
}
