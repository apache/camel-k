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

func TestRunGlobalInstall(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns, "--global").Execute()).Should(BeNil())

		// NS2
		withNewTestNamespace(t, func(ns2 string) {
			Expect(kamel("install", "-n", ns2, "--skip-operator-setup", "--olm", "false").Execute()).Should(BeNil())

			Expect(kamel("run", "-n", ns2, "files/Java.java").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns2, "java"), 5*time.Minute).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns2, "java"), 1*time.Minute).Should(ContainSubstring("Magicstring!"))
			Expect(kamel("delete", "--all", "-n", ns2).Execute()).Should(BeNil())
		})

		Expect(kamel("uninstall", "-n", ns, "--skip-crd", "--skip-cluster-roles").Execute()).Should(BeNil())
	})
}
