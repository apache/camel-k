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

package common

import (
	"reflect"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func TestRunCronExample(t *testing.T) {
	ok, err := kubernetes.IsAPIResourceInstalled(TestClient(), batchv1.SchemeGroupVersion.String(), reflect.TypeOf(batchv1.CronJob{}).Name())
	assert.Nil(t, err)

	if !ok {
		t.Skip("This test requires CronJob batch/v1 API installed.")
		return
	}

	WithNewTestNamespace(t, func(ns string) {
		operatorID := "camel-k-trait-cronjob"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		t.Run("cron", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/cron.yaml").Execute()).To(Succeed())
			Eventually(IntegrationCronJob(ns, "cron"), TestTimeoutMedium).ShouldNot(BeNil())
			Eventually(IntegrationConditionStatus(ns, "cron", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-yaml", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/cron-yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationCronJob(ns, "cron-yaml"), TestTimeoutMedium).ShouldNot(BeNil())
			Eventually(IntegrationConditionStatus(ns, "cron-yaml", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-yaml"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-timer", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/cron-timer.yaml").Execute()).To(Succeed())
			Eventually(IntegrationCronJob(ns, "cron-timer"), TestTimeoutMedium).ShouldNot(BeNil())
			Eventually(IntegrationConditionStatus(ns, "cron-timer", v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-timer"), TestTimeoutMedium).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-fallback", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/cron-fallback.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "cron-fallback"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "cron-fallback", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-fallback"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("cron-quartz", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/cron-quartz.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "cron-quartz"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "cron-quartz", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "cron-quartz"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
