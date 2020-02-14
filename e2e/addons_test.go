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

package e2e

import (
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestAddons(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("master addon", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(kamel("run", "-n", ns, "files/Master.java", "-t", `master.label-key=""`).Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "master"), 5*time.Minute).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "master"), 1*time.Minute).Should(ContainSubstring("Magicstring!"))
			Eventually(configMap(ns, "master-lock"), 30*time.Second).ShouldNot(BeNil())
			// Start a second integration with the same lock (it should not start the route)
			Expect(kamel("run", "-n", ns, "files/Master.java", "--name", "second",
				"-t", "master.configmap=master-lock", "-t", `master.label-key=""`).Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "second"), 5*time.Minute).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "second"), 1*time.Minute).Should(ContainSubstring("started in"))
			Eventually(integrationLogs(ns, "second"), 30*time.Second).ShouldNot(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

	})
}
