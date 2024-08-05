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

package olm

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"testing"

	. "github.com/apache/camel-k/v2/e2e/support"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const installCatalogSourceName = "test-camel-k-source"

func TestOLMInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		os.Setenv("CAMEL_K_TEST_MAKE_DIR", "../../../")
		// Set the configuration required for OLM to work with local catalog
		// Build Camel K local bundle
		ExpectExecSucceed(t, g,
			Make(t, "bundle-build"),
		)
		// Build Operator local index
		ExpectExecSucceed(t, g,
			Make(t, "bundle-index"),
		)

		// This workaround is required because of
		// https://github.com/operator-framework/operator-lifecycle-manager/issues/903#issuecomment-1779366630
		cmd := exec.Command("docker", "inspect", "--format={{.Id}}", "apache/camel-k-bundle-index:2.4.0-SNAPSHOT")
		out, err := cmd.Output()
		require.NoError(t, err)
		newBundleIndexSha := strings.ReplaceAll(string(out), "\n", "")
		g.Expect(newBundleIndexSha).To(Not(Equal("")))

		newBundleIndex := fmt.Sprintf("apache/camel-k-bundle-index@%s", newBundleIndexSha)
		g.Expect(CreateOrUpdateCatalogSource(t, ctx, ns, installCatalogSourceName, newBundleIndex)).To(Succeed())

		ocp, err := openshift.IsOpenShift(TestClient(t))
		require.NoError(t, err)

		if ocp {
			// Wait for pull secret to be created in namespace
			// eg. test-camel-k-source-dockercfg-zlltn
			secretPrefix := fmt.Sprintf("%s-dockercfg-", installCatalogSourceName)
			g.Eventually(SecretByName(t, ctx, ns, secretPrefix), TestTimeoutLong).Should(Not(BeNil()))
		}

		g.Eventually(CatalogSourcePodRunning(t, ctx, ns, installCatalogSourceName), TestTimeoutMedium).Should(BeNil())
		g.Eventually(CatalogSourcePhase(t, ctx, ns, installCatalogSourceName), TestTimeoutLong).Should(Equal("READY"))

		// Install via OLM subscription (should use the latest default available channel)
		ExpectExecSucceed(t, g,
			exec.Command(
				"kubectl",
				"apply",
				"-f",
				"https://operatorhub.io/install/camel-k/camel-k.yaml",
			),
		)

		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		g.Eventually(ClusterServiceVersionPhase(t, ctx, noAdditionalConditions, ns), TestTimeoutMedium).Should(Equal(olm.CSVPhaseSucceeded))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		csvVersion := ClusterServiceVersion(t, ctx, noAdditionalConditions, ns)().Spec.Version
		ipVersionPrefix := fmt.Sprintf("%d.%d", csvVersion.Version.Major, csvVersion.Version.Minor)
		t.Logf("CSV Version installed: %s", csvVersion.Version.String())

		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(OperatorImage(t, ctx, ns), TestTimeoutShort).Should(Equal(defaults.OperatorImage()))

		// Check the IntegrationPlatform has been reconciled
		g.Eventually(PlatformVersion(t, ctx, ns)).Should(ContainSubstring(ipVersionPrefix))

		// Check if restricted security context has been applyed
		operatorPod := OperatorPod(t, ctx, ns)()
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.RunAsNonRoot).To(Equal(kubernetes.DefaultOperatorSecurityContext().RunAsNonRoot))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.Capabilities).To(Equal(kubernetes.DefaultOperatorSecurityContext().Capabilities))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.SeccompProfile).To(Equal(kubernetes.DefaultOperatorSecurityContext().SeccompProfile))
		g.Expect(operatorPod.Spec.Containers[0].SecurityContext.AllowPrivilegeEscalation).To(Equal(kubernetes.DefaultOperatorSecurityContext().AllowPrivilegeEscalation))
	})
}
