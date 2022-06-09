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
	"io/ioutil"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
)

/*
 * TODO
 * Despite the kit and integration being correctly built and the
 * integration phase changed to running, no pod is being created.
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestOpenAPI(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-openapi"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		Expect(KamelRunWithID(operatorID, ns,
			"--name", "petstore",
			"--open-api", "file:files/openapi/petstore-api.yaml",
			"files/openapi/petstore.groovy",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, "petstore"), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		Eventually(Deployment(ns, "petstore"), TestTimeoutLong).
			Should(Not(BeNil()))

		Eventually(IntegrationLogs(ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started listPets (rest://get:/v1:/pets)"))
		Eventually(IntegrationLogs(ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started createPets (rest://post:/v1:/pets)"))
		Eventually(IntegrationLogs(ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started showPetById (rest://get:/v1:/pets/%7BpetId%7D)"))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

/*
 * TODO
 * Despite the kit and integration being correctly built and the
 * integration phase changed to running, no pod is being created.
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestOpenAPIConfigmap(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-openapi-configmap"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		openapiContent, err := ioutil.ReadFile("./files/openapi/petstore-api.yaml")
		assert.Nil(t, err)
		var cmDataProps = make(map[string]string)
		cmDataProps["petstore-api.yaml"] = string(openapiContent)
		NewPlainTextConfigmap(ns, "my-openapi", cmDataProps)

		Expect(KamelRunWithID(operatorID, ns,
			"--name", "petstore",
			"--open-api", "configmap:my-openapi",
			"files/openapi/petstore.groovy",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, "petstore"), TestTimeoutLong).
			Should(Equal(corev1.PodRunning))
		Eventually(Deployment(ns, "petstore"), TestTimeoutLong).
			Should(Not(BeNil()))

		Eventually(IntegrationLogs(ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started listPets (rest://get:/v1:/pets)"))
		Eventually(IntegrationLogs(ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started createPets (rest://post:/v1:/pets)"))
		Eventually(IntegrationLogs(ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started showPetById (rest://get:/v1:/pets/%7BpetId%7D)"))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
