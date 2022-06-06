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
	"os"
	"testing"

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
		Expect(KamelInstall(ns).Execute()).To(Succeed())

		t.Run("Readiness condition with stopped route", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"-t", "health.enabled=true",
				// Enable Jolokia for the test to stop the Camel route
				"-t", "jolokia.enabled=true",
				"-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
			).Execute()).To(Succeed())

			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPhase(ns, "java"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, "java")()

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
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
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
			Eventually(IntegrationCondition(ns, "java", v1.IntegrationConditionReady), TestTimeoutLong).Should(And(
				WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionRuntimeNotReadyReason)),
				WithTransform(IntegrationConditionMessage, HavePrefix(fmt.Sprintf("[Pod %s runtime is not ready: map[route.context.name:camel-1 route.id:route1 route.status:Stopped]", pod.Name))),
			))
			// Check the Integration is still in running phase
			Eventually(IntegrationPhase(ns, "java"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))

			// Clean-up
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
