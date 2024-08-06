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

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestUpgrade(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// We start the test by installing previous version operator
		lastVersion, ok := os.LookupEnv("LAST_RELEASED_VERSION")
		g.Expect(ok).To(BeTrue())
		lastVersionDir := fmt.Sprintf("/tmp/camel-k-v-%s", lastVersion)
		// We clone and install the previous installed operator
		// from source with tag
		ExpectExecSucceed(t, g,
			exec.Command(
				"rm",
				"-rf",
				lastVersionDir,
			))
		ExpectExecSucceed(t, g,
			exec.Command(
				"git",
				"clone",
				"https://github.com/apache/camel-k.git",
				lastVersionDir,
			))
		checkoutCmd := exec.Command(
			"git",
			"checkout",
			fmt.Sprintf("v%s", lastVersion),
		)
		checkoutCmd.Dir = lastVersionDir
		ExpectExecSucceed(t, g, checkoutCmd)
		installPrevCmd := exec.Command(
			"make",
			"install-k8s-global",
			fmt.Sprintf("NAMESPACE=%s", ns),
		)
		installPrevCmd.Dir = lastVersionDir
		ExpectExecSucceed(t, g, installPrevCmd)

		// Check the operator image is the previous one
		g.Eventually(OperatorImage(t, ctx, ns)).Should(ContainSubstring(lastVersion))
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		// Check the IntegrationPlatform has been reconciled
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(lastVersion))

		// We need a different namespace from the global operator
		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsIntegration string) {
			// Run the Integration
			name := RandomizedSuffixName("yaml")
			g.Expect(Kamel(t, ctx, "run", "-n", nsIntegration, "--name", name, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, nsIntegration, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady)).Should(Equal(corev1.ConditionTrue))
			// Check the Integration version
			g.Eventually(IntegrationVersion(t, ctx, nsIntegration, name)).Should(Equal(lastVersion))

			// Let's upgrade the operator with the newer installation
			ExpectExecSucceed(t, g,
				exec.Command(
					"make",
					"install-k8s-global",
					fmt.Sprintf("NAMESPACE=%s", ns),
				),
			)

			// Check the operator image is the current built one
			g.Eventually(OperatorImage(t, ctx, ns)).Should(ContainSubstring(defaults.Version))
			// Check the operator pod is running
			g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			// Check the IntegrationPlatform has been reconciled
			g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
			g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(defaults.Version))

			// Check the Integration hasn't been upgraded
			g.Consistently(IntegrationVersion(t, ctx, nsIntegration, name), 15*time.Second, 3*time.Second).Should(Equal(lastVersion))
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
			g.Eventually(Kits(t, ctx, ns, KitWithVersion(lastVersion))).Should(HaveLen(1))
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
