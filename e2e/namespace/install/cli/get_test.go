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
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelCLIGet(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-cli-get"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("get integration", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			// regex is used for the compatibility of tests between OC and vanilla K8
			// kamel get may have different output depending og the platform
			kitName := Integration(ns, "yaml")().Status.IntegrationKit.Name
			regex := fmt.Sprintf("^NAME\tPHASE\tKIT\n\\s*yaml\tRunning\t(%s/%s|%s)", ns, kitName, kitName)
			Expect(GetOutputString(Kamel("get", "-n", ns))).To(MatchRegexp(regex))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("get several integrations", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "../files/yaml.yaml").Execute()).To(Succeed())
			Expect(KamelRunWithID(operatorID, ns, "../files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			kitName1 := Integration(ns, "java")().Status.IntegrationKit.Name
			kitName2 := Integration(ns, "yaml")().Status.IntegrationKit.Name
			regex := fmt.Sprintf("^NAME\tPHASE\tKIT\n\\s*java\tRunning\t"+
				"(%s/%s|%s)\n\\s*yaml\tRunning\t(%s/%s|%s)\n", ns, kitName1, kitName1, ns, kitName2, kitName2)
			Expect(GetOutputString(Kamel("get", "-n", ns))).To(MatchRegexp(regex))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("get no integrations", func(t *testing.T) {
			Expect(GetOutputString(Kamel("get", "-n", ns))).NotTo(ContainSubstring("Running"))
			Expect(GetOutputString(Kamel("get", "-n", ns))).NotTo(ContainSubstring("Building Kit"))
		})
	})
}
