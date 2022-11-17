//go:build integration && stable
// +build integration,stable

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
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

/*
 * TODO
 * Test has issues on OCP4. See TODO comment in-test for details.
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestHealthTrait(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-health"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("Readiness condition with stopped route", func(t *testing.T) {
			name := "java"
			Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
				"-t", "health.enabled=true",
				// Enable Jolokia for the test to stop the Camel route
				"-t", "jolokia.enabled=true",
				"-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, name)()

			// Stop the Camel route
			request := map[string]string{
				"type":      "exec",
				"mbean":     "org.apache.camel:context=camel-1,name=\"route1\",type=routes",
				"operation": "stop()",
			}
			body, err := json.Marshal(request)
			Expect(err).To(BeNil())

			response, err := TestClient().CoreV1().RESTClient().Post().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/jolokia/", ns, pod.Name)).
				Body(body).
				DoRaw(TestContext)
			Expect(err).To(BeNil())
			Expect(response).To(ContainSubstring(`"status":200`))

			// Check the ready condition has turned falsy
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
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
			Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, HavePrefix(
					fmt.Sprintf(
						"[Pod %s runtime is not ready: map[check.kind:READINESS route.id:route1 route.status:Stopped]",
						pod.Name))),
			))
			// Check the Integration is still in running phase
			Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))

			// Clean-up
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Readiness condition with never ready route", func(t *testing.T) {
			name := "never-ready"
			Expect(KamelRunWithID(operatorID, ns, "files/NeverReady.java",
				"-t", "health.enabled=true",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			Consistently(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), 1*time.Minute).
				Should(Equal(corev1.ConditionFalse))
			Eventually(IntegrationPhase(ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))

			pod := IntegrationPod(ns, name)()

			// Check that the error message is propagated from health checks even if deployment never becomes ready
			Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionErrorReason)),
				WithTransform(IntegrationConditionMessage, HavePrefix(
					fmt.Sprintf(
						"[Pod %s runtime is not ready: map[check.kind:READINESS route.id:never-ready route.status:Stopped]",
						pod.Name))),
			))

			// Clean-up
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
