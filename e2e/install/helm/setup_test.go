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

package helm

import (
	"fmt"
	"os"
	"os/exec"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	. "github.com/onsi/gomega"
)

func TestHelmInstallRunUninstall(t *testing.T) {
	RegisterTestingT(t)

	KAMEL_INSTALL_REGISTRY := os.Getenv("KAMEL_INSTALL_REGISTRY")
	customImage := fmt.Sprintf("%s/apache/camel-k", KAMEL_INSTALL_REGISTRY)

	os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")

	WithNewTestNamespace(t, func(ns string) {
		ExpectExecSucceed(t, Make(fmt.Sprintf("CUSTOM_IMAGE=%s", customImage), "set-version"))
		ExpectExecSucceed(t, Make("release-helm"))
		ExpectExecSucceed(t,
			exec.Command(
				"helm",
				"install",
				"camel-k",
				fmt.Sprintf("../../../docs/charts/camel-k-%s.tgz", defaults.Version),
				"--set",
				fmt.Sprintf("platform.build.registry.address=%s", KAMEL_INSTALL_REGISTRY),
				"--set",
				"platform.build.registry.insecure=true",
				"-n",
				ns,
			),
		)

		Eventually(OperatorPod(ns)).ShouldNot(BeNil())

		// Check if restricted security context has been applyed
		operatorPod := OperatorPod(ns)()
		Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot))
		Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities))
		Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile))
		Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation))

		//Test a simple route
		t.Run("simple route", func(t *testing.T) {
			name := RandomizedSuffixName("yaml")
			Expect(KamelRun(ns, "files/yaml.yaml", "--name", name).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		})

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
	})
}
