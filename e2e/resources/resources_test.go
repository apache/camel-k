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

package resources

import (
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunResourceExamples(t *testing.T) {

	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("run java", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "./files/ResourcesText.java", "--resource", "./files/resources-data.txt").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "resources-text"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-text", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-text"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run java", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "./files/ResourcesBinary.java",
				"--resource", "./files/resources-data.zip", "-d", "camel-zipfile").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "resources-binary"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-binary", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-binary"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})
	})
}
