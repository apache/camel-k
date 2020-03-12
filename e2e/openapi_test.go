// +build knative

// To enable compilation of this file in Goland, go to "Settings -> Go -> Vendoring & Build Tags -> Custom Tags" and add "knative"

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

package e2e

import (
	"testing"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestOpenAPIService(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns, "--trait-profile", string(v1.TraitProfileKnative)).Execute()).Should(BeNil())
		Expect(kamel(
			"run",
			"-n", ns,
			"--name", "petstore",
			"--open-api", "files/petstore-api.yaml",
			"files/petstore.groovy",
		).Execute()).Should(BeNil())

		Eventually(knativeService(ns, "petstore"), testTimeoutLong).
			Should(Not(BeNil()))

		Eventually(integrationLogs(ns, "petstore"), testTimeoutMedium).
			Should(ContainSubstring("Route: listPets started and consuming from: http://0.0.0.0:8080/v1/pets"))
		Eventually(integrationLogs(ns, "petstore"), testTimeoutMedium).
			Should(ContainSubstring("Route: createPets started and consuming from: http://0.0.0.0:8080/v1/pets"))
		Eventually(integrationLogs(ns, "petstore"), testTimeoutMedium).
			Should(ContainSubstring("Route: showPetById started and consuming from: http://0.0.0.0:8080/v1/pets"))

		Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}

func TestOpenAPIDeployment(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns, "--trait-profile", string(v1.TraitProfileKubernetes)).Execute()).Should(BeNil())
		Expect(kamel(
			"run",
			"-n", ns,
			"--name", "petstore",
			"--open-api", "files/petstore-api.yaml",
			"files/petstore.groovy",
		).Execute()).Should(BeNil())

		Eventually(integrationPodPhase(ns, "petstore"), testTimeoutLong).
			Should(Equal(corev1.PodRunning))
		Eventually(deployment(ns, "petstore"), testTimeoutLong).
			Should(Not(BeNil()))

		Eventually(integrationLogs(ns, "petstore"), testTimeoutMedium).
			Should(ContainSubstring("Route: listPets started and consuming from: http://0.0.0.0:8080/v1/pets"))
		Eventually(integrationLogs(ns, "petstore"), testTimeoutMedium).
			Should(ContainSubstring("Route: createPets started and consuming from: http://0.0.0.0:8080/v1/pets"))
		Eventually(integrationLogs(ns, "petstore"), testTimeoutMedium).
			Should(ContainSubstring("Route: showPetById started and consuming from: http://0.0.0.0:8080/v1/pets"))

		Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}
