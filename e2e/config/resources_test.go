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

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	v1 "k8s.io/api/core/v1"

	"io/ioutil"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/gzip"
)

func TestRunResourceExamples(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("Plain text resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resources-route.groovy", "--resource", "./files/resources-data.txt").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resources-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Binary (zip) resource file", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "./files/resources-binary-route.groovy", "--resource", "./files/resources-data.zip", "-d", "camel-zipfile").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resources-binary-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-binary-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-binary-route"), TestTimeoutShort).Should(ContainSubstring("the file body"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("Base64 compressed binary resource file", func(t *testing.T) {
			// We calculate the expected content
			source, err := ioutil.ReadFile("./files/resources-data.txt")
			assert.Nil(t, err)
			expectedBytes, err := gzip.CompressBase64([]byte(source))
			assert.Nil(t, err)

			Expect(Kamel("run", "-n", ns, "./files/resources-base64-encoded-route.groovy", "--resource", "./files/resources-data.txt", "--compression=true").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "resources-base64-encoded-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "resources-base64-encoded-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "resources-base64-encoded-route"), TestTimeoutShort).Should(ContainSubstring(string(expectedBytes)))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
