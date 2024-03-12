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
	"errors"
	"fmt"
	"os"
	"os/exec"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	kerrors "k8s.io/apimachinery/pkg/api/errors"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

// WARNING: this test is not OLM specific but needs certain setting we provide in OLM installation scenario
func TestHelmOperatorUpgrade(t *testing.T) {
	g := NewWithT(t)

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

	if len(CRDs(t)()) > 0 {
		// Clean up old installation - maybe leftover from another test
		if err := UninstallAll(t); err != nil && !kerrors.IsNotFound(err) {
			t.Error(err)
			t.FailNow()
		}
	}
	g.Eventually(CRDs(t), TestTimeoutMedium).Should(HaveLen(0))

	WithNewTestNamespace(t, func(g *WithT, ns string) {
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

		g.Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		g.Eventually(OperatorImage(t, ns)).Should(ContainSubstring(releaseVersion))
		g.Eventually(CRDs(t)).Should(HaveLen(GetExpectedCRDs(releaseVersion)))

		// Test a simple route
		t.Run("simple route", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			g.Expect(KamelRun(t, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		// Delete CRDs with kustomize
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

		g.Eventually(OperatorPod(t, ns)).ShouldNot(BeNil())
		g.Eventually(OperatorImage(t, ns)).Should(ContainSubstring(defaults.Version))

		// Test again a simple route
		t.Run("simple route upgraded", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			g.Expect(KamelRun(t, ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		// Clean up
		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())

		// Delete Integration Platform as it does not get removed with uninstall and might cause next tests to fail
		DeletePlatform(t, ns)()

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
		g.Eventually(OperatorPod(t, ns)).Should(BeNil())

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
