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

package upgrade

import (
	"context"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// WARNING: this test is not OLM specific but needs certain setting we provide in OLM installation scenario
func TestCLIOperatorUpgrade(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		version, ok := os.LookupEnv("KAMEL_K_TEST_RELEASE_VERSION")
		g.Expect(ok).To(BeTrue())

		image, ok := os.LookupEnv("KAMEL_K_TEST_OPERATOR_CURRENT_IMAGE")
		g.Expect(ok).To(BeTrue())

		kamel, ok := os.LookupEnv("RELEASED_KAMEL_BIN")
		g.Expect(ok).To(BeTrue())

		// Set KAMEL_BIN only for this test - don't override the ENV variable for all tests
		g.Expect(os.Setenv("KAMEL_BIN", kamel)).To(Succeed())

		if len(CRDs(t)()) > 0 {
			// Clean up old installation - maybe leftover from another test
			if err := UninstallAll(t, ctx); err != nil && !kerrors.IsNotFound(err) {
				t.Error(err)
				t.FailNow()
			}
		}
		g.Eventually(CRDs(t)).Should(HaveLen(0))

		// Should both install the CRDs and kamel in the given namespace
		g.Expect(Kamel(t, ctx, "install", "-n", ns, "--olm=false", "--force", "--base-image", defaults.BaseImage()).Execute()).To(Succeed())

		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		// Check the IntegrationPlatform has been reconciled
		g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(version))

		// Run the Integration
		name := RandomizedSuffixName("yaml")
		g.Expect(Kamel(t, ctx, "run", "-n", ns, "--name", name, "files/yaml.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))

		// Check the Integration version
		g.Eventually(IntegrationVersion(t, ctx, ns, name)).Should(Equal(version))

		// Clear the KAMEL_BIN environment variable so that the current version is used from now on
		g.Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

		// Upgrade the operator by installing the current version
		g.Expect(Kamel(t, ctx, "install", "-n", ns, "--olm=false", "--skip-default-kamelets-setup", "--force", "--operator-image", image, "--base-image", defaults.BaseImage()).Execute()).To(Succeed())

		// Check the operator image is the current built one
		g.Eventually(OperatorImage(t, ctx, ns)).Should(Equal(image))
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		// Check the IntegrationPlatform has been reconciled
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(defaults.Version))
		// Check the Integration Pod is not rolling a new Pod automatically
		// This is extremely important as we don't want an upgrade to restart any Integration, unless specified by the user
		var numberOfPods = func(pods *int32) bool {
			return *pods == 1
		}
		g.Consistently(IntegrationPodsNumbers(t, ctx, ns, name), 1*time.Minute, 1*time.Second).Should(Satisfy(numberOfPods))
		// Check the Integration hasn't been upgraded
		g.Consistently(IntegrationVersion(t, ctx, ns, name), 5*time.Second, 1*time.Second).Should(Equal(version))

		// Force the Integration upgrade
		g.Expect(Kamel(t, ctx, "rebuild", name, "-n", ns).Execute()).To(Succeed())

		// A catalog should be created with the new configuration
		g.Eventually(DefaultCamelCatalogPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
		// Check the Integration version has been upgraded
		g.Eventually(IntegrationVersion(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(defaults.Version))

		// Check the previous kit is not garbage collected
		g.Eventually(Kits(t, ctx, ns, KitWithVersion(version))).Should(HaveLen(1))
		// Check a new kit is created with the current version
		g.Eventually(Kits(t, ctx, ns, KitWithVersion(defaults.Version))).Should(HaveLen(1))
		// Check the new kit is ready
		g.Eventually(Kits(t, ctx, ns, KitWithVersion(defaults.Version), KitWithPhase(v1.IntegrationKitPhaseReady)),
			TestTimeoutMedium).Should(HaveLen(1))

		kit := Kits(t, ctx, ns, KitWithVersion(defaults.Version))()[0]

		// Check the Integration uses the new image
		g.Eventually(IntegrationKit(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(kit.Name))
		// Check the Integration Pod uses the new kit
		g.Eventually(IntegrationPodImage(t, ctx, ns, name)).Should(Equal(kit.Status.Image))

		// Check the Integration runs correctly
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Clean up
		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		// Delete Integration Platform as it does not get removed with uninstall and might cause next tests to fail
		DeletePlatform(t, ctx, ns)()
		g.Expect(Kamel(t, ctx, "uninstall", "--all", "-n", ns, "--olm=false").Execute()).To(Succeed())
	})
}
