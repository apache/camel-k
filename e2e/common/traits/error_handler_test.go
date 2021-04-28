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

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestErrorHandlerTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("Run errored integration with error handler", func(t *testing.T) {
			name := "error-handler"
			Expect(Kamel("run", "-n", ns, "files/ErroredRoute.java",
				"--name", name,
				"-t", "error-handler.enabled=true",
				"-t", "error-handler.ref=defaultErrorHandler",
				"-p", "camel.beans.defaultErrorHandler=#class:org.apache.camel.builder.DeadLetterChannelBuilder",
				"-p", "camel.beans.defaultErrorHandler.deadLetterUri=log:my-special-error-handler-in-place?level=ERROR&showCaughtException=false&showBody=false&showBodyType=false&showExchangePattern=false",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).ShouldNot(ContainSubstring("InvalidPayloadException"))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("my-special-error-handler-in-place"))
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
