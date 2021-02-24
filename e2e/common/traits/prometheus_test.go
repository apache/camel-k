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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestPrometheusTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)

		// suppress Service Monitor for the time being as CI test runs on OCP 3.11
		createServiceMonitor := false

		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		Expect(Kamel("run", "-n", ns, "../files/Java.java",
			"-t", "prometheus.enabled=true",
			"-t", fmt.Sprintf("prometheus.service-monitor=%v", createServiceMonitor)).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		t.Run("Metrics endpoint works", func(t *testing.T) {
			pod := IntegrationPod(ns, "java")
			response, err := TestClient().CoreV1().RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/q/metrics", ns, pod().Name)).DoRaw(TestContext)
			if err != nil {
				assert.Fail(t, err.Error())
			}
			assert.Contains(t, string(response), "camel.route.exchanges.total")
		})

		t.Run("Service is created", func(t *testing.T) {
			// service name is "<integration name>-prometheus"
			service := Service(ns, "java-prometheus")
			Eventually(service, TestTimeoutShort).ShouldNot(BeNil())
		})

		if ocp && createServiceMonitor {
			t.Run("Service Monitor is created on OpenShift", func(t *testing.T) {
				sm := serviceMonitor(ns, "java")
				Eventually(sm, TestTimeoutShort).ShouldNot(BeNil())
			})
		}

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func serviceMonitor(ns string, name string) func() *monitoringv1.ServiceMonitor {
	return func() *monitoringv1.ServiceMonitor {
		sm := monitoringv1.ServiceMonitor{}
		key := k8sclient.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient().Get(TestContext, key, &sm)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &sm
	}
}
