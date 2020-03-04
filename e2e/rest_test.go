// +build integration

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
	"fmt"
	"net/http"
	"testing"
	"bytes"

	"github.com/apache/camel-k/pkg/util/openshift"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	v1 "k8s.io/api/core/v1"
)

func TestRunREST(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		var profile string
		ocp, err := openshift.IsOpenShift(testClient)
		assert.Nil(t, err)
		if ocp {
			profile = "OpenShift"
		} else {
			profile = "Kubernetes"
		}

		Expect(kamel("install", "-n", ns, "--trait-profile", profile).Execute()).Should(BeNil())
		Expect(kamel("run", "-n", ns, "files/RestConsumer.java", "-d", "camel:undertow").Execute()).Should(BeNil())
		Eventually(integrationPodPhase(ns, "rest-consumer"), testTimeoutMedium).Should(Equal(v1.PodRunning))

		t.Run("Service works", func(t *testing.T) {
			name := "John"
			service := service(ns, "rest-consumer")
			Eventually(service, testTimeoutShort).ShouldNot(BeNil())
			Expect(kamel("run", "-n", ns, "files/RestProducer.groovy", "-p", "serviceName=rest-consumer", "-p", "name="+name).Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "rest-producer"), testTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "rest-consumer"), testTimeoutShort).Should(ContainSubstring(fmt.Sprintf("get %s", name)))
			Eventually(integrationLogs(ns, "rest-producer"), testTimeoutShort).Should(ContainSubstring(fmt.Sprintf("%s Doe", name)))
		})

		if ocp {
			t.Run("Route works", func(t *testing.T) {
				name := "Peter"
				route := route(ns, "rest-consumer")
				Eventually(route, testTimeoutShort).ShouldNot(BeNil())
				response := httpReqest(t, fmt.Sprintf("http://%s/customers/%s", route().Spec.Host, name))
				assert.Equal(t, fmt.Sprintf("%s Doe", name), response)
				Eventually(integrationLogs(ns, "rest-consumer"), testTimeoutShort).Should(ContainSubstring(fmt.Sprintf("get %s", name)))

			})
		}

		// Cleanup
		Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}

func httpReqest(t *testing.T, url string) string {
	response, err := http.Get(url)
	defer response.Body.Close()
	assert.Nil(t, err)

	buf := new(bytes.Buffer)
	buf.ReadFrom(response.Body)
	return buf.String()
}
