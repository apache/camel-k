// +build integration

// To enable compilation of this file in Goland, go to "File -> Settings -> Go -> Build Tags & Vendoring -> Build Tags -> Custom tags" and add "integration"

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

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestJolokiaTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())

		t.Run("Run Java with Jolokia", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"-t", "jolokia.enabled=true",
				"-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
				"-t", "jolokia.extended-client-check=false").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(ns, "java")
			response, err := TestClient().CoreV1().RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/jolokia/", ns, pod().Name)).DoRaw(TestContext)
			if err != nil {
				assert.Fail(t, err.Error())
			}
			assert.Contains(t, string(response), `"status":200`)

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})
	})
}
