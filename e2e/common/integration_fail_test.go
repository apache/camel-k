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
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	v1 "k8s.io/api/core/v1"
)

func TestBadRouteIntegration(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		t.Run("run bad java route", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/BadRoute.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "bad-route"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationPhase(ns, "bad-route"), TestTimeoutShort).Should(Equal(camelv1.IntegrationPhaseError))
			Eventually(IntegrationCondition(ns, "bad-route", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionFalse))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
