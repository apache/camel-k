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
	"fmt"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestPrometheusTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)

		// Do not create PodMonitor for the time being as CI test runs on OCP 3.11
		createPodMonitor := false

		Expect(KamelInstall(ns).Execute()).To(Succeed())

		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"-t", "prometheus.enabled=true",
			"-t", fmt.Sprintf("prometheus.pod-monitor=%v", createPodMonitor)).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
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

		if ocp && createPodMonitor {
			t.Run("PodMonitor is created", func(t *testing.T) {
				sm := podMonitor(ns, "java")
				Eventually(sm, TestTimeoutShort).ShouldNot(BeNil())
			})
		}

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func podMonitor(ns string, name string) func() *monitoringv1.PodMonitor {
	return func() *monitoringv1.PodMonitor {
		pm := monitoringv1.PodMonitor{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient().Get(TestContext, key, &pm)
		if err != nil && errors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &pm
	}
}
