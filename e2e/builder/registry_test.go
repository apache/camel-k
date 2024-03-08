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

package builder

import (
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

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-docker-hub"
		g.Expect(KamelInstallWithID(t, operatorID, ns,
			"--registry", "docker.io",
			"--organization", user,
			"--registry-auth-username", user,
			"--registry-auth-password", pass,
			"--cluster-type", "kubernetes").
			Execute()).To(Succeed())

		g.Expect(KamelRunWithID(t, operatorID, ns, "files/groovy.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ns, "groovy"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ns, "groovy"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(IntegrationPodImage(t, ns, "groovy"), TestTimeoutShort).Should(HavePrefix("docker.io"))

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
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

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-github-registry"
		g.Expect(KamelInstallWithID(t, operatorID, ns,
			"--registry", "docker.pkg.github.com",
			"--organization", repo,
			"--registry-auth-username", user,
			"--registry-auth-password", pass,
			"--cluster-type", "kubernetes").
			Execute()).To(Succeed())

		g.Expect(KamelRunWithID(t, operatorID, ns, "files/groovy.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ns, "groovy"), TestTimeoutLong).Should(Equal(v1.PodRunning))
		g.Eventually(IntegrationLogs(t, ns, "groovy"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		g.Eventually(IntegrationPodImage(t, ns, "groovy"), TestTimeoutShort).Should(HavePrefix("docker.pkg.github.com"))

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
