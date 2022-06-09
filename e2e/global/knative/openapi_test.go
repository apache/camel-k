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

package knative

import (
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestOpenAPIService(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-openapi-service"
		Expect(KamelInstallWithID(operatorID, ns, "--trait-profile", string(v1.TraitProfileKnative)).Execute()).To(Succeed())
		Expect(KamelRunWithID(operatorID, ns,
			"--name", "petstore",
			"--open-api", "file:files/petstore-api.yaml",
			"files/petstore.groovy",
		).Execute()).To(Succeed())

		Eventually(KnativeService(ns, "petstore"), TestTimeoutLong).
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

func TestOpenAPIDeployment(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-openapi-deployment"
		Expect(KamelInstallWithID(operatorID, ns, "--trait-profile", string(v1.TraitProfileKubernetes)).Execute()).To(Succeed())
		Expect(KamelRunWithID(operatorID, ns,
			"--name", "petstore",
			"--open-api", "file:files/petstore-api.yaml",
			"files/petstore.groovy",
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
