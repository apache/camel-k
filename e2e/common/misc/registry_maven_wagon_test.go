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

package misc

import (
	"context"
	"fmt"
	"path/filepath"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

func TestImageRegistryIsAMavenRepository(t *testing.T) {
	t.Parallel()

	ocp, err := openshift.IsOpenShift(TestClient(t))
	require.NoError(t, err)
	if ocp {
		t.Skip("Avoid running on OpenShift until CA and secret are injected client side")
		return
	}

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-registry-maven-repo"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("image registry is a maven repository", func(t *testing.T) {
			// Create integration that should decrypt an encrypted message to "foobar" and log it
			name := RandomizedSuffixName("foobar-decryption")
			jar, err := filepath.Abs("files/registry/sample-decryption-1.0.jar?skipPOM=true")
			require.NoError(t, err)
			pom, err := filepath.Abs("files/registry/sample-decryption-1.0.pom")
			require.NoError(t, err)

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/registry/FoobarDecryption.java",
				"--name", name,
				"-d", fmt.Sprintf("file://%s", jar),
				"-d", fmt.Sprintf("file://%s", pom)).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("foobar"))
		})

		t.Run("local files are mounted in the integration container at the default path", func(t *testing.T) {
			name := RandomizedSuffixName("laughing-route-default-path")

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/registry/LaughingRoute.java",
				"--name", name,
				"-p", "location=/deployments/?filename=laugh.txt",
				"-d", "file://files/registry/laugh.txt").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("haha"))
		})

		t.Run("local files are mounted in the integration container at a custom path", func(t *testing.T) {
			name := RandomizedSuffixName("laughing-route-custom-path")
			customPath := "this/is/a/custom/path/"

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/registry/LaughingRoute.java",
				"--name", name,
				"-p", fmt.Sprintf("location=%s", customPath),
				"-d", fmt.Sprintf("file://files/registry/laugh.txt?targetPath=%slaugh.txt", customPath)).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("haha"))
		})

		t.Run("local directory is mounted in the integration container", func(t *testing.T) {
			name := RandomizedSuffixName("laughing-route-directory")

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/registry/LaughingRoute.java",
				"--name", name,
				"-p", "location=files/registry/",
				"-d", fmt.Sprintf("file://files/registry/laughs/?targetPath=files/registry/")).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("haha"))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("hehe"))
		})

		t.Run("pom file is extracted from JAR", func(t *testing.T) {
			// Create integration that should decrypt foobar and log it
			name := RandomizedSuffixName("foobar-decryption-pom-extraction")
			jar, err := filepath.Abs("files/registry/sample-decryption-1.0.jar")
			require.NoError(t, err)

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/registry/FoobarDecryption.java",
				"--name", name,
				"-d", fmt.Sprintf("file://%s", jar)).Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("foobar"))
		})

		t.Run("dependency can be used at build time", func(t *testing.T) {
			// Create integration that should run a Xslt transformation whose template needs to be present at build time
			name := RandomizedSuffixName("xslt")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/registry/classpath/Xslt.java",
				"--name", name,
				"-d", "file://files/registry/classpath/cheese.xsl?targetPath=xslt/cheese.xsl&classpath=true").Execute()).To(Succeed())

			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("<cheese><item>A</item></cheese>"))
		})

		// Clean up
		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
