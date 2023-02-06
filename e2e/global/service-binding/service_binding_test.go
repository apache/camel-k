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

package service_binding

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestServiceBindingTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-service-binding"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		// Create our mock service config
		host := "hostname"
		port := "12324"
		service := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mock-service-config",
				Namespace: ns,
				Annotations: map[string]string{
					"service.binding/host": "path={.data.service-host}",
					"service.binding/port": "path={.data.service-port}",
				},
			},
			Data: map[string]string{
				"service-host": host,
				"service-port": port,
			},
		}
		serviceRef := fmt.Sprintf("%s:%s/%s", service.TypeMeta.Kind, ns, service.ObjectMeta.Name)
		Expect(TestClient().Create(TestContext, service)).To(Succeed())

		// Create integration and bind it to our service
		name := "service-binding"
		Expect(KamelRunWithID(operatorID, ns, "files/ServiceBinding.java",
			"--name", name,
			"--connect", serviceRef,
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring(fmt.Sprintf("%s:%s", host, port)))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
