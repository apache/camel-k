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
	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"net"
	"testing"
	"time"
)

func TestKamelCLIDebug(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("debug local default port check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Expect(portIsInUse("127.0.0.1", "5005")()).To(BeFalse())

			debugTestContext, cancel := context.WithCancel(TestContext)
			defer cancelAndWait(cancel)
			go KamelWithContext(debugTestContext, "debug", "yaml", "-n", ns).ExecuteContext(debugTestContext)

			Eventually(portIsInUse("127.0.0.1", "5005"), TestTimeoutMedium, 5*time.Second).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug local port check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Expect(portIsInUse("127.0.0.1", "5006")()).To(BeFalse())

			debugTestContext, cancel := context.WithCancel(TestContext)
			defer cancelAndWait(cancel)
			go KamelWithContext(debugTestContext, "debug", "yaml", "--port", "5006", "-n", ns).ExecuteContext(debugTestContext)

			Eventually(portIsInUse("127.0.0.1", "5006"), TestTimeoutMedium, 5*time.Second).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug logs check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			debugTestContext, cancel := context.WithCancel(TestContext)
			defer cancelAndWait(cancel)

			go KamelWithContext(debugTestContext, "debug", "yaml", "-n", ns).ExecuteContext(debugTestContext)

			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutMedium).Should(ContainSubstring("Listening for transport dt_socket at address: 5005"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Pod config test", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			debugTestContext, cancel := context.WithCancel(TestContext)
			defer cancelAndWait(cancel)

			go KamelWithContext(debugTestContext, "debug", "yaml", "-n", ns).ExecuteContext(debugTestContext)

			Eventually(func() string {
				return IntegrationPod(ns, "yaml")().Spec.Containers[0].Args[0]
			}).Should(ContainSubstring("-agentlib:jdwp=transport=dt_socket,server=y,suspend=y,address=*:5005"))

			Expect(IntegrationPod(ns, "yaml")().GetLabels()["camel.apache.org/debug"]).To(Not(BeNil()))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}

func portIsInUse(host string, port string) func() bool {
	return func() bool {

		conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), time.Second*5)

		if conn != nil {
			defer conn.Close()
		}

		return conn != nil && err == nil
	}
}

func cancelAndWait(cancel context.CancelFunc) {
	cancel()
	time.Sleep(TestTimeoutShort)
}
