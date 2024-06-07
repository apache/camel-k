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
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/apache/camel-k/v2/e2e/support"
	testutil "github.com/apache/camel-k/v2/e2e/support/util"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// WARNING: this test is not OLM specific but needs certain setting we provide in OLM installation scenario
func TestKustomizeUpgrade(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// ********* START
		// TODO: we need to replace this CLI based installation with Kustomize installation after 2.4 release
		// this is a workaround as Kustomize was not working properly pre 2.4
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
		g.Expect(Kamel(t, ctx, "install", "-n", ns, "--global").Execute()).To(Succeed())
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)
		// Check the IntegrationPlatform has been reconciled
		g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(version))
		// TODO: replace the code above
		// ************* END

		// We need a different namespace from the global operator
		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsIntegration string) {
			// Run the Integration
			name := RandomizedSuffixName("yaml")
			g.Expect(Kamel(t, ctx, "run", "-n", nsIntegration, "--name", name, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, nsIntegration, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady)).Should(Equal(corev1.ConditionTrue))
			// Check the Integration version
			g.Eventually(IntegrationVersion(t, ctx, nsIntegration, name)).Should(Equal(version))

			// Clear the KAMEL_BIN environment variable so that the current version is used from now on
			g.Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

			// Upgrade the operator by installing the current version
			registry := os.Getenv("KIND_REGISTRY")
			kustomizeDir := testutil.MakeTempCopyDir(t, "../../../install")
			g.Expect(registry).NotTo(Equal(""))
			// We must change a few values in the Kustomize config
			ExpectExecSucceed(t, g,
				exec.Command(
					"sed",
					"-i",
					fmt.Sprintf("s/namespace: .*/namespace: %s/", ns),
					fmt.Sprintf("%s/overlays/kubernetes/descoped/kustomization.yaml", kustomizeDir),
				))
			ExpectExecSucceed(t, g,
				exec.Command(
					"sed",
					"-i",
					fmt.Sprintf("s/address: .*/address: %s/", registry),
					fmt.Sprintf("%s/overlays/kubernetes/descoped/integration-platform.yaml", kustomizeDir),
				))

			ExpectExecSucceed(t, g, Kubectl(
				"apply",
				"-k",
				fmt.Sprintf("%s/overlays/kubernetes/descoped", kustomizeDir),
				"--server-side",
				"--force-conflicts",
			))

			// Check the operator image is the current built one
			g.Eventually(OperatorImage(t, ctx, ns)).Should(Equal(image))
			// Check the operator pod is running
			g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			// Check the IntegrationPlatform has been reconciled
			g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
			g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(defaults.Version))

			// Check the Integration hasn't been upgraded
			g.Consistently(IntegrationVersion(t, ctx, nsIntegration, name), 15*time.Second, 3*time.Second).Should(Equal(version))
			// Make sure that any Pod rollout is completing successfully
			// otherwise we are probably in front of a non breaking compatibility change
			g.Consistently(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady),
				2*time.Minute, 15*time.Second).Should(Equal(corev1.ConditionTrue))

			// Force the Integration upgrade
			g.Expect(Kamel(t, ctx, "rebuild", name, "-n", nsIntegration).Execute()).To(Succeed())

			// A catalog should be created with the new configuration
			g.Eventually(DefaultCamelCatalogPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
			// Check the Integration version has been upgraded
			g.Eventually(IntegrationVersion(t, ctx, nsIntegration, name), TestTimeoutMedium).Should(Equal(defaults.Version))

			// Check the previous kit is not garbage collected
			g.Eventually(Kits(t, ctx, ns, KitWithVersion(version))).Should(HaveLen(1))
			// Check a new kit is created with the current version
			g.Eventually(Kits(t, ctx, ns, KitWithVersion(defaults.Version))).Should(HaveLen(1))
			// Check the new kit is ready
			g.Eventually(Kits(t, ctx, ns, KitWithVersion(defaults.Version), KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutMedium).Should(HaveLen(1))

			kit := Kits(t, ctx, ns, KitWithVersion(defaults.Version))()[0]

			// Check the Integration uses the new image
			g.Eventually(IntegrationKit(t, ctx, nsIntegration, name), TestTimeoutMedium).Should(Equal(kit.Name))
			// Check the Integration Pod uses the new kit
			g.Eventually(IntegrationPodImage(t, ctx, nsIntegration, name)).Should(Equal(kit.Status.Image))

			// Check the Integration runs correctly
			g.Eventually(IntegrationPodPhase(t, ctx, nsIntegration, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		})
		// TODO: we should verify new CRDs installed are the same as the one defined in the source core here
	})
}
