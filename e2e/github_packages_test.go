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

package e2e

import (
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	v1 "k8s.io/api/core/v1"
)

func TestRunWithGithubPackagesRegistry(t *testing.T) {
	user := os.Getenv("TEST_GITHUB_PACKAGES_USERNAME")
	pass := os.Getenv("TEST_GITHUB_PACKAGES_PASSWORD")
	repo := os.Getenv("TEST_GITHUB_PACKAGES_REPO")
	if user == "" || pass == "" || repo == "" {
		t.Skip("no github packages data: skipping")
	} else {
		withNewTestNamespace(t, func(ns string) {
			Expect(kamel("install",
				"-n", ns,
				"--registry", "docker.pkg.github.com",
				"--organization", repo,
				"--registry-auth-username", user,
				"--registry-auth-password", pass,
				"--cluster-type", "kubernetes").
				Execute()).Should(BeNil())

			Expect(kamel("run", "-n", ns, "files/groovy.groovy").Execute()).Should(BeNil())
			Eventually(integrationPodPhase(ns, "groovy"), 5*time.Minute).Should(Equal(v1.PodRunning))
			Eventually(integrationLogs(ns, "groovy"), 1*time.Minute).Should(ContainSubstring("Magicstring!"))
			Eventually(integrationPodImage(ns, "groovy"), 1*time.Minute).Should(HavePrefix("docker.pkg.github.com"))

			Expect(kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})
	}

}
