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
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
)

func TestKamelCLIDump(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		t.Run("dump empty namespace", func(t *testing.T) {
			dump := GetOutputString(Kamel("dump", "-n", ns))

			Expect(dump).To(ContainSubstring("Found 0 integrations:"))
			Expect(dump).To(ContainSubstring("Found 0 deployments:"))
		})

		t.Run("dump non-empty namespace", func(t *testing.T) {
			Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
			Expect(Kamel("run", "files/yaml.yaml", "-n", ns).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "yaml")).Should(ContainSubstring("Magicstring!"))

			dump := GetOutputString(Kamel("dump", "-n", ns))

			Expect(dump).To(ContainSubstring("Found 1 platforms"))
			Expect(dump).To(ContainSubstring("Found 1 integrations"))
			Expect(dump).To(ContainSubstring("name: yaml"))
			Expect(dump).To(ContainSubstring("Magicstring!"))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
