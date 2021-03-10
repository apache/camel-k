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
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func TestOperatorUpgrade(t *testing.T) {
	// Clean all cluster-wide resources that could corrupt the test run
	Expect(Kamel("uninstall", "--all", "--olm=false").Execute()).To(Succeed())

	WithNewTestNamespace(t, func(ns string) {
		version, ok := os.LookupEnv("KAMEL_K_TEST_RELEASE_VERSION")
		Expect(ok).To(BeTrue())

		image, ok := os.LookupEnv("KAMEL_K_TEST_OPERATOR_CURRENT_IMAGE")
		Expect(ok).To(BeTrue())

		kamel, ok := os.LookupEnv("RELEASED_KAMEL_BIN")
		Expect(ok).To(BeTrue())

		//set KAMEL_BIN only for this test - don't override the ENV variable for all tests
		Expect(os.Setenv("KAMEL_BIN", kamel)).To(Succeed())

		Expect(Kamel("install", "--olm=false", "--cluster-setup", "--force").Execute()).To(Succeed())
		Expect(Kamel("install", "--olm=false", "-n", ns).Execute()).To(Succeed())

		// Check the operator pod is running
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(v1.PodRunning))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		// Check the IntegrationPlatform has been reconciled
		Eventually(PlatformVersion(ns), TestTimeoutMedium).Should(Equal(version))

		// Run the Integration
		name := "yaml"
		Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))
		// Check the Integration version
		Eventually(IntegrationVersion(ns, "yaml")).Should(Equal(version))
		kit := IntegrationKit(ns, "yaml")()

		// Clear the KAMEL_BIN environment variable so that the current version is used from now on
		Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

		// Upgrade the operator by installing the current version
		// FIXME: it seems forcing the installation does not re-install the CRDs
		Expect(Kamel("install", "--olm=false", "--cluster-setup", "--force").Execute()).To(Succeed())
		Expect(Kamel("install", "-n", ns, "--olm=false", "--force", "--operator-image", image).Execute()).To(Succeed())

		// Check the operator image is the current built one
		Eventually(OperatorImage(ns)).Should(Equal(image))
		// Check the operator pod is running
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		// Check the IntegrationPlatform has been reconciled
		Eventually(PlatformVersion(ns), TestTimeoutMedium).Should(Equal(defaults.Version))

		// Check the Integration hasn't been upgraded
		Consistently(IntegrationVersion(ns, "yaml"), 3*time.Second).Should(Equal(version))

		// Force the Integration upgrade
		Expect(Kamel("rebuild", "yaml", "-n", ns).Execute()).To(Succeed())

		// Check the Integration version change
		Eventually(IntegrationVersion(ns, "yaml")).Should(Equal(defaults.Version))
		// Check the previous kit is not garbage collected
		Eventually(KitsWithVersion(ns, version)).Should(Equal(1))
		// Check a new kit is created with the current version
		Eventually(KitsWithVersion(ns, defaults.Version)).Should(Equal(1))
		// Check the Integration uses the new kit
		Eventually(IntegrationKit(ns, "yaml"), TestTimeoutMedium).ShouldNot(Equal(kit))
		// Check the Integration runs correctly
		Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(v1.ConditionTrue))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("uninstall", "--all", "--olm=false").Execute()).To(Succeed())
	})
}
