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
)

func TestKameletServiceBindingTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--operator-image-pull-policy", "Always").Execute()).To(Succeed())

		// Create our mock service config
		message := "hello"
		service := &corev1.ConfigMap{
			TypeMeta: metav1.TypeMeta{
				Kind:       "ConfigMap",
				APIVersion: "v1",
			},
			ObjectMeta: metav1.ObjectMeta{
				Name:      "mock-service-config",
				Namespace: ns,
				Annotations: map[string]string{
					"service.binding/message": "path={.data.message}",
				},
			},
			Data: map[string]string{
				"message": message,
			},
		}
		serviceRef := fmt.Sprintf("%s:%s/%s", service.TypeMeta.Kind, ns, service.ObjectMeta.Name)
		Expect(TestClient().Create(TestContext, service)).To(Succeed())

		Expect(CreateTimerKamelet(ns, "timer-source")()).To(Succeed())

		Expect(Kamel("bind", "timer-source", "log:info", "--connect", serviceRef, "-n", ns).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "timer-source-to-log"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

		Eventually(IntegrationLogs(ns, "timer-source-to-log")).Should(ContainSubstring("Body: hello"))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
