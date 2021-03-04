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

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestRunCronExample(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("cron", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/cron.yaml").Execute()).To(Succeed())
			Eventually(IntegrationCronJob(ns, "cron"), TestTimeoutMedium).ShouldNot(BeNil())
			Eventually(IntegrationCondition(ns, "cron", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-timer", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/cron-timer.yaml").Execute()).To(Succeed())
			Eventually(IntegrationCronJob(ns, "cron-timer"), TestTimeoutMedium).ShouldNot(BeNil())
			Eventually(IntegrationCondition(ns, "cron-timer", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-timer"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-fallback", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/cron-fallback.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "cron-fallback"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "cron-fallback", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-fallback"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-quartz", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/cron-quartz.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "cron-quartz"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
			Eventually(IntegrationCondition(ns, "cron-quartz", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-quartz"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
