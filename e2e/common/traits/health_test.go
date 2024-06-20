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

package traits

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/onsi/gomega/gstruct"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	. "github.com/onsi/gomega"

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
				"-t", "health.enabled=true",
				"-t", "jolokia.enabled=true", "-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
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

			for i, pod := range pods {
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
					Body(body).
					DoRaw(ctx)
				g.Expect(err).To(BeNil())
				g.Expect(response).To(ContainSubstring(`"status":200`))

				g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionFalse))
				g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
					WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
					WithTransform(IntegrationConditionMessage, Equal(fmt.Sprintf("%d/3 pods are not ready", i+1)))))

				t.Logf("Route on integration pod %s/%s stopped", pod.Namespace, pod.Name)
			}

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *v1.IntegrationCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 3 {
						return false
					}

					var r *v1.HealthCheckResponse

					for _, pod := range c.Pods {
						for h := range pod.Health {
							if pod.Health[h].Name == "camel-routes" {
								r = &pod.Health[h]
							}
						}

						if r == nil {
							return false
						}

						if r.Data == nil {
							return false
						}

						var data map[string]interface{}
						if err := json.Unmarshal(r.Data, &data); err != nil {
							return false
						}
						if data["check.kind"].(string) != "READINESS" || data["route.status"].(string) != "Stopped" {
							return false
						}
					}
					return true
				}))

			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		})

		t.Run("Readiness condition with stopped route", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
				"-t", "health.enabled=true",
				"-t", "jolokia.enabled=true", "-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
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
				Body(body).
				DoRaw(ctx)
			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned false
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
			// And it contains details about the runtime state

			//
			// TODO
			// Integration has different runtime state reporting on OCP4
			//
			// lastProbeTime: null
			// lastTransitionTime: "2021-12-08T20:12:14Z"
			// message: 'containers with unready status: [integration]'
			// reason: ContainersNotReady
			// status: False
			// type: Ready
			//
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *v1.IntegrationCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 1 {
						return false
					}

					var r *v1.HealthCheckResponse

					for h := range c.Pods[0].Health {
						if c.Pods[0].Health[h].Name == "camel-routes" {
							r = &c.Pods[0].Health[h]
						}
					}

					if r == nil {
						return false
					}

					if r.Data == nil {
						return false
					}

					var data map[string]interface{}
					if err := json.Unmarshal(r.Data, &data); err != nil {
						return false
					}

					return data["check.kind"].(string) == "READINESS" && data["route.status"].(string) == "Stopped"
				}))

			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		})

		t.Run("Readiness condition with stopped binding", func(t *testing.T) {
			name := RandomizedSuffixName("stopped-binding")
			source := RandomizedSuffixName("my-health-timer-source")
			sink := RandomizedSuffixName("my-health-log-sink")

			g.Expect(CreateTimerKamelet(t, ctx, ns, source)()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ctx, ns, sink)()).To(Succeed())

			g.Expect(KamelBind(t, ctx, ns, source, sink, "-p",
				"source.message=Magicstring!", "-p", "sink.loggerName=binding",
				"--trait", "health.enabled=true",
				"--trait", "jolokia.enabled=true",
				"--trait", "jolokia.use-ssl-client-authentication=false",
				"--trait", "jolokia.protocol=http",
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
				Body(body).
				DoRaw(ctx)

			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned false
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
			// And it contains details about the runtime state

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *v1.IntegrationCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 1 {
						return false
					}

					var r *v1.HealthCheckResponse

					for h := range c.Pods[0].Health {
						if c.Pods[0].Health[h].Name == "camel-routes" {
							r = &c.Pods[0].Health[h]
						}
					}

					if r == nil {
						return false
					}

					if r.Data == nil {
						return false
					}

					var data map[string]interface{}
					if err := json.Unmarshal(r.Data, &data); err != nil {
						return false
					}

					return data["check.kind"].(string) == "READINESS" && data["route.status"].(string) == "Stopped" && data["route.id"].(string) == "binding"
				}))

			g.Eventually(PipeCondition(t, ctx, ns, name, camelv1.PipeConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *camelv1.PipeCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 1 {
						return false
					}

					var r *v1.HealthCheckResponse

					for h := range c.Pods[0].Health {
						if c.Pods[0].Health[h].Name == "camel-routes" {
							r = &c.Pods[0].Health[h]
						}
					}

					if r == nil {
						return false
					}

					if r.Data == nil {
						return false
					}

					var data map[string]interface{}
					if err := json.Unmarshal(r.Data, &data); err != nil {
						return false
					}

					return data["check.kind"].(string) == "READINESS" && data["route.status"].(string) == "Stopped" && data["route.id"].(string) == "binding"
				}))
		})

		t.Run("Readiness condition with never ready route", func(t *testing.T) {
			name := RandomizedSuffixName("never-ready")

			g.Expect(KamelRun(t, ctx, ns, "files/NeverReady.java", "--name", name,
				"-t", "health.enabled=true",
				"-p", "camel.health.routesEnabled=false",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Consistently(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), 1*time.Minute).
				Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))

			// Check that the error message is propagated from health checks even if deployment never becomes ready
			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *v1.IntegrationCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 1 {
						return false
					}

					var r *v1.HealthCheckResponse

					for h := range c.Pods[0].Health {
						if c.Pods[0].Health[h].Name == "never-ready" {
							r = &c.Pods[0].Health[h]
						}
					}

					if r == nil {
						return false
					}

					if r.Data == nil {
						return false
					}

					var data map[string]interface{}
					if err := json.Unmarshal(r.Data, &data); err != nil {
						return false
					}

					return r.Status == v1.HealthCheckStatusDown && data["check.kind"].(string) == "READINESS"
				}))
		})

		t.Run("Startup condition with never ready route", func(t *testing.T) {
			name := RandomizedSuffixName("startup-probe-never-ready-route")

			g.Expect(KamelRun(t, ctx, ns, "files/NeverReady.java", "--name", name,
				"-t", "health.enabled=true",
				"-t", "health.startup-probe-enabled=true", "-t", "health.startup-timeout=60").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
			g.Consistently(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), 1*time.Minute).Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *v1.IntegrationCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 1 {
						return false
					}

					var r *v1.HealthCheckResponse

					for h := range c.Pods[0].Health {
						if c.Pods[0].Health[h].Name == "never-ready" && c.Pods[0].Health[h].Status == "DOWN" {
							r = &c.Pods[0].Health[h]
						}
					}

					if r == nil {
						return false
					}

					if r.Data == nil {
						return false
					}

					var data map[string]interface{}
					if err := json.Unmarshal(r.Data, &data); err != nil {
						return false
					}

					return data["check.kind"].(string) == "READINESS"
				}))
		})

		t.Run("Startup condition with ready route", func(t *testing.T) {
			name := RandomizedSuffixName("startup-probe-ready-route")

			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name,
				"-t", "health.enabled=true",
				"-t", "health.startup-probe-enabled=true", "-t", "health.startup-timeout=60").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))

			g.Eventually(IntegrationCondition(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionDeploymentReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 ready replicas"))))
		})
	})
}
