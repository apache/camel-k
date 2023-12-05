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

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

// WARNING: this test is not OLM specific but needs certain setting we provide in OLM installation scenario
func TestHelmOperatorUpgrade(t *testing.T) {
	RegisterTestingT(t)

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

	os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")

	// Ensure no CRDs are already installed
	UninstallAll()
	Eventually(CRDs()).Should(HaveLen(0))

	WithNewTestNamespace(t, func(ns string) {

		// Install operator in last released version
		ExpectExecSucceed(t,
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

		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(OperatorImage(ns)).Should(ContainSubstring(releaseVersion))
		Eventually(CRDs()).Should(HaveLen(ExpectedCRDs))

		//Test a simple route
		t.Run("simple route", func(t *testing.T) {
			name := "simpleyaml"
			Expect(KamelRun(ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		// Upgrade CRDs with kustomize
		ExpectExecSucceed(t,
			exec.Command(
				"kubectl",
				"replace",
				"-f",
				"../../../helm/camel-k/crds/",
				"-n",
				ns,
			),
		)

		// Upgrade operator to current version
		ExpectExecSucceed(t, Make(fmt.Sprintf("CUSTOM_IMAGE=%s", customImage), "set-version"))
		ExpectExecSucceed(t, Make("release-helm"))
		ExpectExecSucceed(t,
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

		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(OperatorImage(ns)).Should(ContainSubstring(defaults.Version))

		//Test again a simple route
		t.Run("simple route upgraded", func(t *testing.T) {
			name := "upgradedyaml"
			Expect(KamelRun(ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

		// Uninstall with helm
		ExpectExecSucceed(t,
			exec.Command(
				"helm",
				"uninstall",
				"camel-k",
				"-n",
				ns,
			),
		)
		Eventually(OperatorPod(ns)).Should(BeNil())

		//  helm does not remove the CRDs
		Eventually(CRDs()).Should(HaveLen(ExpectedCRDs))
		ExpectExecSucceed(t,
			exec.Command(
				"kubectl",
				"delete",
				"-k",
				"../../../config/crd/",
				"-n",
				ns,
			),
		)
		Eventually(CRDs()).Should(HaveLen(0))
	})
}
