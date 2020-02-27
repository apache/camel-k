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

package e2e

import (
	"testing"
	"time"

	"github.com/apache/camel-k/pkg/util/defaults"
	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestPlatformUpgrade(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(platformVersion(ns)).Should(Equal(defaults.Version))

		// Scale the operator down to zero
		Eventually(scaleOperator(ns, 0), 10*time.Second).Should(BeNil())
		Eventually(operatorPod(ns)).Should(BeNil())

		// Change the version to an older one
		Eventually(setPlatformVersion(ns, "an.older.one")).Should(BeNil())
		Eventually(platformVersion(ns)).Should(Equal("an.older.one"))

		// Scale the operator up
		Eventually(scaleOperator(ns, 1)).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())

		// Check the platform version change
		Eventually(platformVersion(ns)).Should(Equal(defaults.Version))
	})
}

func TestIntegrationUpgrade(t *testing.T) {
	withNewTestNamespace(t, func(ns string) {
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(platformVersion(ns)).Should(Equal(defaults.Version))

		// Run an integration
		Expect(kamel("run", "-n", ns, "files/js.js").Execute()).Should(BeNil())
		Eventually(integrationPodPhase(ns, "js"), testTimeoutMedium).Should(Equal(v1.PodRunning))
		initialImage := integrationPodImage(ns, "js")()

		// Scale the operator down to zero
		Eventually(scaleOperator(ns, 0)).Should(BeNil())
		Eventually(operatorPod(ns)).Should(BeNil())

		// Change the version to an older one
		Expect(setIntegrationVersion(ns, "js", "an.older.one")).Should(BeNil())
		Expect(setAllKitsVersion(ns, "an.older.one")).Should(BeNil())
		Eventually(integrationVersion(ns, "js")).Should(Equal("an.older.one"))
		Eventually(kitsWithVersion(ns, "an.older.one")).Should(Equal(1))
		Eventually(kitsWithVersion(ns, defaults.Version)).Should(Equal(0))

		// Scale the operator up
		Eventually(scaleOperator(ns, 1)).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
		Eventually(operatorPodPhase(ns)).Should(Equal(v1.PodRunning))

		// No auto-update expected
		Consistently(integrationVersion(ns, "js"), 3*time.Second).Should(Equal("an.older.one"))

		// Clear the integration status
		Expect(kamel("rebuild", "js", "-n", ns).Execute()).Should(BeNil())

		// Check the integration version change
		Eventually(integrationVersion(ns, "js")).Should(Equal(defaults.Version))
		Eventually(kitsWithVersion(ns, "an.older.one")).Should(Equal(1)) // old one is not recycled
		Eventually(kitsWithVersion(ns, defaults.Version)).Should(Equal(1))
		Eventually(integrationPodImage(ns, "js"), testTimeoutMedium).ShouldNot(Equal(initialImage)) // rolling deployment triggered
		Eventually(integrationPodPhase(ns, "js"), testTimeoutMedium).Should(Equal(v1.PodRunning))
	})
}
