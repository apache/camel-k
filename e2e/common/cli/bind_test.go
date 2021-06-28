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
	v1 "k8s.io/api/core/v1"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelCLIBind(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(CreateTimerKamelet(ns, "test-timer-source")()).To(Succeed())

		t.Run("bind timer to log", func(t *testing.T) {
			Expect(Kamel("bind", "test-timer-source", "log:info", "-p", "source.message=helloTest", "-n", ns).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "test-timer-source-to-log"), TestTimeoutLong).Should(Equal(v1.PodRunning))

			Eventually(IntegrationLogs(ns, "test-timer-source-to-log")).Should(ContainSubstring("Body: helloTest"))
			Expect(Kamel("bind", "test-timer-source", "log:info", "-p", "source.message=newText", "-n", ns).Execute()).To(Succeed())
			Eventually(IntegrationLogs(ns, "test-timer-source-to-log")).Should(ContainSubstring("Body: newText"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("unsuccessful binding, no property", func(t *testing.T) {
			Expect(Kamel("bind", "timer-source", "log:info", "-n", ns).Execute()).NotTo(Succeed())
		})
	})
}
