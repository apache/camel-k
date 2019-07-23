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
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunServiceCombo(t *testing.T) {
	withNewTestNamespace(func(ns string) {
		RegisterTestingT(t)
		Expect(kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).Should(BeNil())
		Expect(kamel("run", "-n", ns, "files/knative2.groovy").Execute()).Should(BeNil())
		Expect(kamel("run", "-n", ns, "files/knative1.groovy").Execute()).Should(BeNil())
		Eventually(integrationPodPhase(ns, "knative2"), 10*time.Minute).Should(Equal(v1.PodRunning))
		Eventually(integrationPodPhase(ns, "knative1"), 10*time.Minute).Should(Equal(v1.PodRunning))
		Eventually(integrationLogs(ns, "knative1"), 5*time.Minute).Should(ContainSubstring("Received: Hello from knative2"))
		Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}

func TestRunChannelCombo(t *testing.T) {
	withNewTestNamespace(func(ns string) {
		RegisterTestingT(t)
		Expect(createKnativeChannel(ns, "messages")()).Should(BeNil())
		Expect(kamel("install", "-n", ns, "--trait-profile", "knative").Execute()).Should(BeNil())
		Expect(kamel("run", "-n", ns, "files/knativech2.groovy").Execute()).Should(BeNil())
		Expect(kamel("run", "-n", ns, "files/knativech1.groovy").Execute()).Should(BeNil())
		Eventually(integrationPodPhase(ns, "knativech2"), 10*time.Minute).Should(Equal(v1.PodRunning))
		Eventually(integrationPodPhase(ns, "knativech1"), 10*time.Minute).Should(Equal(v1.PodRunning))
		Eventually(integrationLogs(ns, "knativech2"), 5*time.Minute).Should(ContainSubstring("Received: Hello from knativech1"))
		Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
	})
}
