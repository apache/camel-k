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
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, operatorNs string) {
		// Let's make sure no CRD is yet available in the cluster
		// as we must make the procedure to install them accordingly
		g.Expect(CRDs(t)()).Should(BeNil(), "No Camel K CRDs should be previously installed for this test")
		// We start the test by installing previous version operator
		lastVersion, ok := os.LookupEnv("LAST_RELEASED_VERSION")
		g.Expect(ok).To(BeTrue(), "Missing last released version: you need to set it into LAST_RELEASED_VERSION env var")

		// Install previous version: mind that the registry configuration has to be stored by the action
		// and expected in camel namespace
		applyCmd := exec.Command(
			"kubectl",
			"apply",
			"-k",
			"github.com/apache/camel-k/install/overlays/all-namespaces?ref=v"+lastVersion,
			"-n",
			"camel",
			"--server-side",
			"--force-conflicts",
		)
		ExpectExecSucceed(t, g, applyCmd)

		// Check the operator image is the previous one
		g.Eventually(OperatorImage(t, ctx, operatorNs)).Should(ContainSubstring(lastVersion))
		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, operatorNs), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// We need a different namespace from the global operator
		WithNewTestNamespace(t, func(ctx context.Context, g *WithT, nsIntegration string) {
			// Run the Integration
			name := RandomizedSuffixName("yaml")
			g.Expect(Kamel(t, ctx, "run", "-n", nsIntegration, "--name", name, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, nsIntegration, name)).Should(Equal(corev1.PodRunning))
			// Check the Integration version
			g.Eventually(IntegrationVersion(t, ctx, nsIntegration, name)).Should(Equal(lastVersion))
			// Get the info of the runtime, as we need for further check later
			lastRuntimeVersion := Integration(t, ctx, nsIntegration, name)().Status.RuntimeVersion

			// Let's upgrade the operator with the newer installation (default in camel namespace)
			installNextCmd := exec.Command(
				"kubectl",
				"apply",
				"-k",
				"install/overlays/all-namespaces",
				"--server-side",
				"--force-conflicts",
			)
			installNextCmd.Dir = "../.."
			ExpectExecSucceed(t, g, installNextCmd)
			// The default installation come with a dev registry. In this test we need to make sure
			// it reuses the common registry used by previous installation, so, we immediately disable the
			// feature
			disableDevRegistryCmd := exec.Command(
				"kubectl",
				"-n",
				"camel",
				"set",
				"env",
				"deployment/camel-k-operator",
				"ENABLE_DEV_REGISTRY=\"false\"",
			)
			disableDevRegistryCmd.Dir = "../.."
			ExpectExecSucceed(t, g, disableDevRegistryCmd)

			// Refresh the test client to account for the newly installed CRDs
			RefreshClient(t)

			// Check the operator image is the current built one
			g.Eventually(OperatorImage(t, ctx, operatorNs)).Should(ContainSubstring(defaults.Version))
			// Check the operator pod is running
			g.Eventually(OperatorPodPhase(t, ctx, operatorNs), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

			// Check the Integration hasn't been upgraded
			g.Consistently(IntegrationVersion(t, ctx, nsIntegration, name), 15*time.Second, 3*time.Second).
				Should(Equal(lastVersion))
			// Make sure that any Pod rollout is completing successfully
			// otherwise we are probably in front of a non breaking compatibility change
			g.Consistently(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady),
				2*time.Minute, 15*time.Second).Should(Equal(corev1.ConditionTrue))

			// Force the Integration upgrade
			g.Expect(Kamel(t, ctx, "rebuild", name, "-n", nsIntegration).Execute()).To(Succeed())

			// Check the Integration version has been upgraded
			g.Eventually(IntegrationVersion(t, ctx, nsIntegration, name), TestTimeoutMedium).Should(Equal(defaults.Version))

			// Check the previous kit is not garbage collected
			g.Eventually(Kits(t, ctx, operatorNs, KitWithRuntimeVersion(lastRuntimeVersion))).Should(HaveLen(1))
			// Check a new kit is created with the current version
			g.Eventually(Kits(t, ctx, operatorNs, KitWithRuntimeVersion(defaults.DefaultRuntimeVersion))).Should(HaveLen(1))
			// Check the new kit is ready
			g.Eventually(Kits(t, ctx, operatorNs, KitWithRuntimeVersion(defaults.DefaultRuntimeVersion), KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutMedium).Should(HaveLen(1))

			kit := Kits(t, ctx, operatorNs, KitWithRuntimeVersion(defaults.DefaultRuntimeVersion))()[0]

			// Check the Integration uses the new image
			g.Eventually(IntegrationKitName(t, ctx, nsIntegration, name), TestTimeoutMedium).Should(Equal(kit.Name))
			// Check the Integration Pod uses the new kit
			g.Eventually(IntegrationPodImage(t, ctx, nsIntegration, name)).Should(Equal(kit.Status.Image))

			// Check the Integration runs correctly
			g.Eventually(IntegrationConditionStatus(t, ctx, nsIntegration, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, nsIntegration, name)).
				Should(Equal(corev1.PodRunning))
		})
	}, "camel")
}
