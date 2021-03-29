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
	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
	"net"
	"strings"
	"testing"
	"time"
)

func TestKamelCLIDebug(t *testing.T){
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("debug local default port check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			Expect(LocalPortIsInUse("127.0.0.1", "5005", time.Second * 5)).To(BeFalse())

			go Kamel("debug", "yaml", "-n", ns).Execute()

			Eventually(func() bool {
				tt := LocalPortIsInUse("127.0.0.1", "5005", time.Second * 5)
				return tt
			}, TestTimeoutLong).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug local port check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(func () bool {
				return LocalPortIsInUse("127.0.0.1", "5010", time.Second * 5)
			}, TestTimeoutShort).Should(BeFalse())

			go Kamel("debug", "yaml", "--port", "5010", "-n", ns).Execute()

			Eventually(func() bool { return LocalPortIsInUse("127.0.0.1", "5010", time.Second * 5)}).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug logs check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			go Kamel("debug", "yaml", "-n", ns).Execute()

			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutMedium).Should(ContainSubstring("Listening for transport dt_socket at address: 5005"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug remote default port check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			go Kamel("debug", "yaml", "-n", ns).Execute()

			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			podIP := IntegrationPod(ns, "yaml")().Status.PodIP
			Eventually(func() bool { return RemotePortIsInUse(podIP, "5005")}).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug remote port check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			go Kamel("debug", "yaml", "--remote-port", "5012", "-n", ns).Execute()

			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			podIP := IntegrationPod(ns, "yaml")().Status.PodIP
			Eventually(func() bool { return RemotePortIsInUse(podIP, "5012")}).Should(BeTrue())

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug flag check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			go Kamel("debug", "yaml", "-n", ns).Execute()

			Eventually(func() string {
				return IntegrationPod(ns, "yaml")().Spec.Containers[0].Args[0]
			}).Should(ContainSubstring("-agentlib:jdwp=transport=dt_socket,server=y,suspend=y,address=*:5005"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("debug label check", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

			go Kamel("debug", "yaml", "-n", ns).Execute()

			Eventually(func() string {
				labelMap := IntegrationPod(ns, "yaml")().GetLabels()
				if val, ok := labelMap["camel.apache.org/debug"]; ok {
					return val
				} else {
					return "false"
				}}).Should(Equal("true"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}

func LocalPortIsInUse(host string, port string, timeout time.Duration) bool {
	conn, err := net.DialTimeout("tcp", net.JoinHostPort(host, port), timeout)

	if err != nil {
		if strings.Contains(err.Error(), "too many open files") {
			time.Sleep(timeout)
			LocalPortIsInUse(host, port, timeout)
		} else {
			return false
		}
		return false
	}

	conn.Close()
	return true
}

func RemotePortIsInUse(host string, port string) bool {
	connection, err := net.DialTimeout("tcp", host + ":" + port, time.Second*5)

	if err != nil {
		return true
	}
	if connection != nil {
		defer connection.Close()
		return false
	}
	return true
}