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
	"os"
	"testing"
	"time"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/util/openshift"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestAddons(t *testing.T) {
	forceMasterTest := os.Getenv("CAMEL_K_FORCE_MASTER_TEST") == "true"
	if !forceMasterTest {
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)
		if ocp {
			t.Skip("Prefer not to run on OpenShift to avoid giving more permissions to the user running tests")
			return
		}
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("master works", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/Master.java").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "master"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationLogs(ns, "master"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// TODO enable check on configmap or lease
			//Eventually(ConfigMap(ns, "master-lock"), 30*time.Second).ShouldNot(BeNil())
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("only one integration with master runs", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/Master.java",
				"--name", "first",
				"--label", "leader-group=same",
				"-t", "master.label-key=leader-group",
				"-t", "master.label-value=same",
				"-t", "owner.target-labels=leader-group").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "first"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationLogs(ns, "first"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// TODO enable check on configmap or lease
			//Eventually(ConfigMap(ns, "first-lock"), 30*time.Second).ShouldNot(BeNil())
			// Start a second integration with the same lock (it should not start the route)
			Expect(Kamel("run", "-n", ns, "files/Master.java",
				"--name", "second",
				"--label", "leader-group=same",
				"-t", "master.label-key=leader-group",
				"-t", "master.label-value=same",
				"-t", "master.configmap=first-lock",
				"-t", "owner.target-labels=leader-group").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "second"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationLogs(ns, "second"), TestTimeoutShort).Should(ContainSubstring("started in"))
			Eventually(IntegrationLogs(ns, "second"), 30*time.Second).ShouldNot(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

	})
}
