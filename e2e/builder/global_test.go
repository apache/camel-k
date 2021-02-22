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

package builder

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestRunGlobalInstall(t *testing.T) {
	forceGlobalTest := os.Getenv("CAMEL_K_FORCE_GLOBAL_TEST") == "true"
	if !forceGlobalTest {
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)
		if ocp {
			t.Skip("Prefer not to run on OpenShift to avoid giving more permissions to the user running tests")
			return
		}
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--global").Execute()).To(Succeed())

		// NS2: namespace without operator
		WithNewTestNamespace(t, func(ns2 string) {
			Expect(Kamel("install", "-n", ns2, "--skip-operator-setup", "--olm=false").Execute()).To(Succeed())

			Expect(Kamel("run", "-n", ns2, "files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns2, "java"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationLogs(ns2, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns2).Execute()).To(Succeed())

			Expect(ConfigMap(ns2, platform.OperatorLockName)()).To(BeNil(), "No locking configmap expected")
		})

		// NS3: namespace with its own operator
		WithNewTestNamespace(t, func(ns3 string) {
			Expect(Kamel("install", "-n", ns3, "--olm=false").Execute()).To(Succeed())

			Expect(Kamel("run", "-n", ns3, "files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns3, "java"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationLogs(ns3, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns3).Execute()).To(Succeed())

			Expect(ConfigMap(ns3, platform.OperatorLockName)()).ShouldNot(BeNil(),
				"OperatorSDK is expected to use configmaps for locking: if this changes (e.g. using Leases) we should update our guard logic",
			)
		})

		Expect(Kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute()).To(Succeed())
	})
}
