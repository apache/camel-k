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
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/onsi/gomega/gstruct"

	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestHealthTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {

		t.Run("Readiness condition with stopped route scaled", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
				"-t", "health.enabled=true",
				// Enable Jolokia for the test to stop the Camel route
				"-t", "jolokia.enabled=true",
				"-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
				"--name", name,
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Expect(ScaleIntegration(t, ns, name, 3)).To(Succeed())
			// Check the readiness condition becomes falsy
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
			// Check the scale cascades into the Deployment scale
			g.Eventually(IntegrationPods(t, ns, name), TestTimeoutShort).Should(HaveLen(3))
			// Check it also cascades into the Integration scale subresource Status field
			g.Eventually(IntegrationStatusReplicas(t, ns, name), TestTimeoutShort).
				Should(gstruct.PointTo(BeNumerically("==", 3)))
			// Finally check the readiness condition becomes truthy back
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ns, name)()
			healthTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "health")
			g.Expect(healthTrait).ToNot(BeNil())
			g.Expect(len(healthTrait)).To(Equal(1))
			g.Expect(healthTrait["enabled"]).To(Equal(true))

			pods := IntegrationPods(t, ns, name)()

			for i, pod := range pods {
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
					DoRaw(TestContext)
				g.Expect(err).To(BeNil())
				g.Expect(response).To(ContainSubstring(`"status":200`))

				g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
					Should(Equal(corev1.ConditionFalse))

				g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
					WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
					WithTransform(IntegrationConditionMessage, Equal(fmt.Sprintf("%d/3 pods are not ready", i+1)))))
			}

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
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

			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))

			// Clean-up
			g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Readiness condition with stopped route", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
				"-t", "health.enabled=true",
				// Enable Jolokia for the test to stop the Camel route
				"-t", "jolokia.enabled=true",
				"-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
				"--name", name,
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ns, name)()

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
				DoRaw(TestContext)
			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned false
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
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
			// status: "False"
			// type: Ready
			//
			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
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

			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))

			// Clean-up
			g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Readiness condition with stopped binding", func(t *testing.T) {
			name := RandomizedSuffixName("stopped-binding")
			source := RandomizedSuffixName("my-health-timer-source")
			sink := RandomizedSuffixName("my-health-log-sink")

			g.Expect(CreateTimerKamelet(t, ns, source)()).To(Succeed())
			g.Expect(CreateLogKamelet(t, ns, sink)()).To(Succeed())

			g.Expect(KamelBindWithID(t, operatorID, ns,
				source,
				sink,
				"-p", "source.message=Magicstring!",
				"-p", "sink.loggerName=binding",
				"--annotation", "trait.camel.apache.org/health.enabled=true",
				"--annotation", "trait.camel.apache.org/jolokia.enabled=true",
				"--annotation", "trait.camel.apache.org/jolokia.use-ssl-client-authentication=false",
				"--annotation", "trait.camel.apache.org/jolokia.protocol=http",
				"--name", name,
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ns, name)()

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
				DoRaw(TestContext)

			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned false
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionFalse))
			// And it contains details about the runtime state

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
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

			g.Eventually(PipeCondition(t, ns, name, camelv1.PipeConditionReady), TestTimeoutLong).Should(
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

			// Clean-up
			g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			g.Expect(DeleteKamelet(t, ns, source)).To(Succeed())
			g.Expect(DeleteKamelet(t, ns, sink)).To(Succeed())
		})

		t.Run("Readiness condition with never ready route", func(t *testing.T) {
			name := RandomizedSuffixName("never-ready")

			g.Expect(KamelRunWithID(t, operatorID, ns, "files/NeverReady.java",
				"--name", name,
				"-t", "health.enabled=true",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			g.Consistently(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), 1*time.Minute).
				Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))

			// Check that the error message is propagated from health checks even if deployment never becomes ready
			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
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

					return data["check.kind"].(string) == "READINESS" && data["route.status"].(string) == "Stopped" && data["route.id"].(string) == "never-ready"
				}))
		})

		t.Run("Startup condition with never ready route", func(t *testing.T) {
			name := RandomizedSuffixName("startup-probe-never-ready-route")

			g.Expect(KamelRunWithID(t, operatorID, ns, "files/NeverReady.java",
				"--name", name,
				"-t", "health.enabled=true",
				"-t", "health.startup-probe-enabled=true",
				"-t", "health.startup-timeout=60",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
			g.Consistently(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), 1*time.Minute).Should(Equal(corev1.ConditionFalse))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 pods are not ready"))))

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(
				Satisfy(func(c *v1.IntegrationCondition) bool {
					if c.Status != corev1.ConditionFalse {
						return false
					}
					if len(c.Pods) != 1 {
						return false
					}

					var r *v1.HealthCheckResponse

					for h := range c.Pods[0].Health {
						if c.Pods[0].Health[h].Name == "camel-routes" && c.Pods[0].Health[h].Status == "DOWN" {
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

					return data["check.kind"].(string) == "READINESS" && data["route.status"].(string) == "Stopped" && data["route.id"].(string) == "never-ready"
				}))

			Satisfy(func(events *corev1.EventList) bool {
				for e := range events.Items {
					if events.Items[e].Type == "Warning" && events.Items[e].Reason == "Unhealthy" && strings.Contains(events.Items[e].Message, "Startup probe failed") {
						return true
					}
				}
				return false
			})
		})

		t.Run("Startup condition with ready route", func(t *testing.T) {
			name := RandomizedSuffixName("startup-probe-ready-route")

			g.Expect(KamelRunWithID(t, operatorID, ns, "files/Java.java",
				"--name", name,
				"-t", "health.enabled=true",
				"-t", "health.startup-probe-enabled=true",
				"-t", "health.startup-timeout=60",
			).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPhase(t, ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))

			g.Eventually(IntegrationCondition(t, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionDeploymentReadyReason)),
				WithTransform(IntegrationConditionMessage, Equal("1/1 ready replicas"))))

			Satisfy(func(is *v1.IntegrationSpec) bool {
				if *is.Traits.Health.Enabled == true && *is.Traits.Health.StartupProbeEnabled == true && is.Traits.Health.StartupTimeout == 60 {
					return true
				}
				return false
			})

		})

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
