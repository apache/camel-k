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

package traits

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestPodTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "pod-template-test"
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/PodTest.groovy",
			"--name", name,
			"--pod-template", "files/template.yaml",
		).Execute()).To(Succeed())

		// check integration is deployed
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// check that integrations is working and reading data created by sidecar container
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Content from the sidecar container"))
		// check that env var is injected
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("hello from the template"))
		pod := IntegrationPod(ns, name)()

		// check if ENV variable is applied
		envValue := getEnvVar("TEST_VARIABLE", pod.Spec)
		Expect(envValue).To(Equal("hello from the template"))
	})
}

func getEnvVar(name string, spec corev1.PodSpec) string {
	for _, i := range spec.Containers[0].Env {
		if i.Name == name {
			return i.Value
		}
	}
	return ""
}
