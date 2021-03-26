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
	. "github.com/apache/camel-k/e2e/support"
	 "github.com/apache/camel-k/pkg/util/defaults"
	v1 "k8s.io/api/core/v1"
)

func TestKamelCliDescribe(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(v1.PodRunning))

 		t.Run("Test kamel describe integration", func(t *testing.T) {
 			integration := GetOutputString(Kamel("describe","integration", "yaml", "-n", ns))
 			Expect(integration).To(ContainSubstring("Name:                yaml"))
			Expect(integration).To(ContainSubstring("Phase:               Running"))
			Expect(integration).To(ContainSubstring("Dependencies:"))
			Expect(integration).To(ContainSubstring("Conditions:"))
 		})

 		t.Run("Test kamel describe integration kit", func(t *testing.T) {
 			kitName := Integration(ns, "yaml")().Status.Kit
			kit := GetOutputString(Kamel("describe","kit", kitName, "-n", ns))
			Expect(kit).To(ContainSubstring("Namespace:           " + ns))
			Expect(kit).To(ContainSubstring("Version:             " + defaults.Version))
			Expect(kit).To(ContainSubstring("camel-quarkus:core"))
			Expect(kit).To(ContainSubstring("Artifacts:"))
			Expect(kit).To(ContainSubstring("Dependencies:"))
 		})

		t.Run("Test kamel describe integration platform", func(t *testing.T) {
			platform := GetOutputString(Kamel("describe","platform", "camel-k", "-n", ns))
 			Expect(platform).To(ContainSubstring("Name:                camel-k"))
 			Expect(platform).To(ContainSubstring("Namespace:           " + ns))
 			Expect(platform).To(ContainSubstring("Version:             " + defaults.Version))
		})
	})
}
