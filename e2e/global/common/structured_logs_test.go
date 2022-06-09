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
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func TestStructuredLogs(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		name := "java"
		operatorID := "camel-k-logging"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"-t", "logging.format=json").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		pod := OperatorPod(ns)()
		Expect(pod).NotTo(BeNil())

		// pod.Namespace could be different from ns if using global operator
		logs := StructuredLogs(pod.Namespace, pod.Name, corev1.PodLogOptions{}, false)
		Expect(logs).NotTo(BeEmpty())

		it := Integration(ns, name)()
		Expect(it).NotTo(BeNil())
		build := Build(ns, it.Status.IntegrationKit.Name)()
		Expect(build).NotTo(BeNil())

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
