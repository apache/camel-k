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

package build

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func TestRunIncrementalBuild(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(KamelInstall(ns).Execute()).To(Succeed())

		name := "java"
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"--name", name,
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName := IntegrationKit(ns, name)()
		Eventually(Kit(ns, integrationKitName)().Status.BaseImage).Should(Equal(defaults.BaseImage()))
		// Another integration that should be built on top of the previous IntegrationKit
		// just add a new random dependency
		name2 := "java2"
		Expect(Kamel("run", "-n", ns, "files/Java.java",
			"--name", name2,
			"-d", "camel-zipfile",
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name2), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name2, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name2), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		integrationKitName2 := IntegrationKit(ns, name2)()
		// the container can come in a format like
		// 10.108.177.66/test-d7cad110-bb1d-4e79-8a0e-ebd44f6fe5d4/camel-k-kit-c8357r4k5tp6fn1idm60@sha256:d49716f0429ad8b23a1b8d20a357d64b1aa42a67c1a2a534ebd4c54cd598a18d
		// we should be save just to check the substring is contained
		Eventually(Kit(ns, integrationKitName2)().Status.BaseImage).Should(ContainSubstring(integrationKitName))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
