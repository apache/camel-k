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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// WARNING: this test is not OLM specific but needs certain setting we provide in OLM installation scenario
func TestHelmOperatorUpgrade(t *testing.T) {
	KAMEL_INSTALL_REGISTRY := os.Getenv("KAMEL_INSTALL_REGISTRY")
	// need to add last release version
	releaseVersion := os.Getenv("KAMEL_K_TEST_RELEASE_VERSION")
	// if the last released version chart is not present skip the test
	releaseChart := fmt.Sprintf("../../../docs/charts/camel-k-%s.tgz", releaseVersion)
	if _, err := os.Stat(releaseChart); errors.Is(err, os.ErrNotExist) {
		t.Skip("last release version chart not found: skipping")
		return
	}

	customImage := fmt.Sprintf("%s/apache/camel-k", KAMEL_INSTALL_REGISTRY)

	if err := os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../"); err != nil {
		t.Logf("Unable to set makefile directory envvar - %s", err.Error())
	}

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Install operator in last released version
		ExpectExecSucceed(t, g,
			exec.Command(
				"helm",
				"install",
				"camel-k",
				releaseChart,
				"--set",
				fmt.Sprintf("platform.build.registry.address=%s", KAMEL_INSTALL_REGISTRY),
				"--set",
				"platform.build.registry.insecure=true",
				"-n",
				ns,
			),
		)

		g.Eventually(OperatorPod(t, ctx, ns)).ShouldNot(BeNil())
		g.Eventually(OperatorImage(t, ctx, ns)).Should(ContainSubstring(releaseVersion))
		g.Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(releaseVersion)))

		// Test a simple route
		t.Run("simple route", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			// Don't do this in a production environment! This is a quick and dirty workaround to bypass the Helm
			// update problem which limit the possibility to upgrade CRDs within the same upgrade process.
			// Running this is breaking the operator which would not able to watch the deleted resources. It's okey for
			// testing though.
			ExpectExecSucceed(t, g,
				exec.Command(
					"kubectl",
					"delete",
					"--ignore-not-found",
					"-f",
					"../../../helm/camel-k/crds/",
					"-n",
					ns,
				),
			)

			// Re-Create CRDs with kustomize
			ExpectExecSucceed(t, g,
				exec.Command(
					"kubectl",
					"create",
					"-f",
					"../../../helm/camel-k/crds/",
					"-n",
					ns,
				),
			)

			// Upgrade operator to current version
			ExpectExecSucceed(t, g, Make(t, fmt.Sprintf("CUSTOM_IMAGE=%s", customImage), "set-version"))
			ExpectExecSucceed(t, g, Make(t, "release-helm"))
			ExpectExecSucceed(t, g,
				exec.Command(
					"helm",
					"upgrade",
					"camel-k",
					fmt.Sprintf("../../../docs/charts/camel-k-%s.tgz", defaults.Version),
					"--set",
					fmt.Sprintf("platform.build.registry.address=%s", KAMEL_INSTALL_REGISTRY),
					"--set",
					"platform.build.registry.insecure=true",
					"-n",
					ns,
					"--force",
				),
			)
			// Check the IntegrationPlatform has been reconciled
			g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
			g.Eventually(PlatformVersion(t, ctx, ns), TestTimeoutMedium).Should(Equal(defaults.Version))
			// Check the Integration hasn't been upgraded
			g.Consistently(IntegrationVersion(t, ctx, ns, name), 5*time.Second, 1*time.Second).Should(Equal(releaseVersion))
			// Force the Integration upgrade
			g.Expect(Kamel(t, ctx, "rebuild", name, "-n", ns).Execute()).To(Succeed())
			// A catalog should be created with the new configuration
			g.Eventually(DefaultCamelCatalogPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.CamelCatalogPhaseReady))
			// Check the Integration version has been upgraded
			g.Eventually(IntegrationVersion(t, ctx, ns, name), TestTimeoutMedium).Should(Equal(defaults.Version))
			// Check the previous kit is not garbage collected
			g.Eventually(Kits(t, ctx, ns, KitWithVersion(releaseVersion))).Should(HaveLen(1))
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
		})

		// Delete Integration Platform as it does not get removed with uninstall and might cause next tests to fail
		DeletePlatform(t, ctx, ns)()

		// Uninstall with helm
		ExpectExecSucceed(t, g,
			exec.Command(
				"helm",
				"uninstall",
				"camel-k",
				"-n",
				ns,
			),
		)
		g.Eventually(OperatorPod(t, ctx, ns)).Should(BeNil())

		//  helm does not remove the CRDs
		g.Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(defaults.Version)))
		ExpectExecSucceed(t, g,
			exec.Command(
				"kubectl",
				"delete",
				"-k",
				"../../../pkg/resources/config/crd/",
				"-n",
				ns,
			),
		)
		g.Eventually(CRDs(t)).Should(HaveLen(0))
	})
}
