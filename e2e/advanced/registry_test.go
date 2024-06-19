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

package advanced

import (
	"context"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	v1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestRunWithDockerHubRegistry(t *testing.T) {
	user := os.Getenv("TEST_DOCKER_HUB_USERNAME")
	pass := os.Getenv("TEST_DOCKER_HUB_PASSWORD")
	if user == "" || pass == "" {
		t.Skip("no docker hub credentials: skipping")
		return
	}

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-docker-hub"
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--registry", "docker.io", "--organization", user, "--registry-auth-username", user, "--registry-auth-password", pass, "--cluster-type", "kubernetes")).To(Succeed())

		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/example.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "example"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "example"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(IntegrationPodImage(t, ctx, ns, "example"), TestTimeoutShort).Should(HavePrefix("docker.io"))

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunWithGithubPackagesRegistry(t *testing.T) {
	user := os.Getenv("TEST_GITHUB_PACKAGES_USERNAME")
	pass := os.Getenv("TEST_GITHUB_PACKAGES_PASSWORD")
	repo := os.Getenv("TEST_GITHUB_PACKAGES_REPO")
	if user == "" || pass == "" || repo == "" {
		t.Skip("no github packages data: skipping")
		return
	}

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-github-registry"
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns, "--registry", "docker.pkg.github.com", "--organization", repo, "--registry-auth-username", user, "--registry-auth-password", pass, "--cluster-type", "kubernetes")).To(Succeed())

		g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/example.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "example"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, "example"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(IntegrationPodImage(t, ctx, ns, "example"), TestTimeoutShort).Should(HavePrefix("docker.pkg.github.com"))

		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
