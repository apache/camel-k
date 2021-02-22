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
	"time"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func TestPlatformUpgrade(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(PlatformVersion(ns)).Should(Equal(defaults.Version))

		// Scale the operator down to zero
		Eventually(ScaleOperator(ns, 0), 10*time.Second).Should(BeNil())
		Eventually(OperatorPod(ns)).Should(BeNil())

		// Change the version to an older one
		Eventually(SetPlatformVersion(ns, "an.older.one")).Should(Succeed())
		Eventually(PlatformVersion(ns)).Should(Equal("an.older.one"))

		// Scale the operator up
		Eventually(ScaleOperator(ns, 1)).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())

		// Check the platform version change
		Eventually(PlatformVersion(ns)).Should(Equal(defaults.Version))
	})
}

func TestIntegrationUpgrade(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Eventually(PlatformVersion(ns)).Should(Equal(defaults.Version))

		// Run an integration
		Expect(Kamel("run", "-n", ns, "files/js.js").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "js"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		initialKit := IntegrationKit(ns, "js")()

		// Scale the operator down to zero
		Eventually(ScaleOperator(ns, 0)).Should(BeNil())
		Eventually(OperatorPod(ns)).Should(BeNil())

		// Change the version to an older one
		Expect(SetIntegrationVersion(ns, "js", "an.older.one")).To(Succeed())
		Expect(SetAllKitsVersion(ns, "an.older.one")).To(Succeed())
		Eventually(IntegrationVersion(ns, "js")).Should(Equal("an.older.one"))
		Eventually(KitsWithVersion(ns, "an.older.one")).Should(Equal(1))
		Eventually(KitsWithVersion(ns, defaults.Version)).Should(Equal(0))

		// Scale the operator up
		Eventually(ScaleOperator(ns, 1)).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(OperatorPodPhase(ns)).Should(Equal(v1.PodRunning))

		// No auto-update expected
		Consistently(IntegrationVersion(ns, "js"), 3*time.Second).Should(Equal("an.older.one"))

		// Clear the integration status
		Expect(Kamel("rebuild", "js", "-n", ns).Execute()).To(Succeed())

		// Check the integration version change
		Eventually(IntegrationVersion(ns, "js")).Should(Equal(defaults.Version))
		Eventually(KitsWithVersion(ns, "an.older.one")).Should(Equal(1)) // old one is not recycled
		Eventually(KitsWithVersion(ns, defaults.Version)).Should(Equal(1))
		Eventually(IntegrationKit(ns, "js"), TestTimeoutMedium).ShouldNot(Equal(initialKit))
		Eventually(IntegrationPodPhase(ns, "js"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
	})
}
