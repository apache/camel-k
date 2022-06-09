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
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func TestOperatorUpgrade(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		version, ok := os.LookupEnv("KAMEL_K_TEST_RELEASE_VERSION")
		Expect(ok).To(BeTrue())

		image, ok := os.LookupEnv("KAMEL_K_TEST_OPERATOR_CURRENT_IMAGE")
		Expect(ok).To(BeTrue())

		kamel, ok := os.LookupEnv("RELEASED_KAMEL_BIN")
		Expect(ok).To(BeTrue())

		// Set KAMEL_BIN only for this test - don't override the ENV variable for all tests
		Expect(os.Setenv("KAMEL_BIN", kamel)).To(Succeed())

		// Should both install the CRDs and kamel in the given namespace
		Expect(Kamel("install", "-n", ns, "--olm=false", "--force").Execute()).To(Succeed())

		// Check the operator pod is running
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		// Check the IntegrationPlatform has been reconciled
		Eventually(PlatformVersion(ns), TestTimeoutMedium).Should(Equal(version))

		// Run the Integration
		name := "yaml"
		Expect(KamelRun(ns, "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))

		// Check the Integration version
		Eventually(IntegrationVersion(ns, name)).Should(Equal(version))

		// Clear the KAMEL_BIN environment variable so that the current version is used from now on
		Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

		// Upgrade the operator by installing the current version
		Expect(KamelInstall(ns, "--olm=false", "--force", "--operator-image", image).Execute()).To(Succeed())

		// Check the operator image is the current built one
		Eventually(OperatorImage(ns)).Should(Equal(image))
		// Check the operator pod is running
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		// Check the IntegrationPlatform has been reconciled
		Eventually(PlatformVersion(ns), TestTimeoutMedium).Should(Equal(defaults.Version))

		// Check the Integration hasn't been upgraded
		Consistently(IntegrationVersion(ns, name), 5*time.Second, 1*time.Second).Should(Equal(version))

		// Force the Integration upgrade
		Expect(Kamel("rebuild", name, "-n", ns).Execute()).To(Succeed())

		// Check the Integration version has been upgraded
		Eventually(IntegrationVersion(ns, name)).Should(Equal(defaults.Version))

		// Check the previous kit is not garbage collected
		Eventually(Kits(ns, KitWithVersion(version))).Should(HaveLen(1))
		// Check a new kit is created with the current version
		Eventually(Kits(ns, KitWithVersion(defaults.Version))).Should(HaveLen(1))
		// Check the new kit is ready
		Eventually(Kits(ns, KitWithVersion(defaults.Version), KitWithPhase(v1.IntegrationKitPhaseReady)),
			TestTimeoutMedium).Should(HaveLen(1))

		kit := Kits(ns, KitWithVersion(defaults.Version))()[0]

		// Check the Integration uses the new image
		Eventually(IntegrationKit(ns, name), TestTimeoutMedium).Should(Equal(kit.Name))
		// Check the Integration Pod uses the new kit
		Eventually(IntegrationPodImage(ns, name)).Should(Equal(kit.Status.Image))

		// Check the Integration runs correctly
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("uninstall", "--all", "-n", ns, "--olm=false").Execute()).To(Succeed())
	})
}
