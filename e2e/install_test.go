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
	"testing"

	. "github.com/onsi/gomega"
)

func TestBasicInstallation(t *testing.T) {
	withNewTestNamespace(func(ns string) {
		RegisterTestingT(t)
		Expect(kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(operatorPod(ns)).ShouldNot(BeNil())
	})
}

func TestAlternativeImageInstallation(t *testing.T) {
	withNewTestNamespace(func(ns string) {
		RegisterTestingT(t)
		Expect(kamel("install", "-n", ns, "--operatorPod-image", "x/y:latest").Execute()).Should(BeNil())
		Eventually(operatorImage(ns)).Should(Equal("x/y:latest"))
	})
}

func TestKitJVMInstallation(t *testing.T) {
	withNewTestNamespace(func(ns string) {
		RegisterTestingT(t)
		Expect(kamel("install", "-n", ns, "--kit", "jvm").Execute()).Should(BeNil())
		Eventually(build(ns, "jvm")).ShouldNot(BeNil())
	})
}

func TestMavenRepositoryInstallation(t *testing.T) {
	withNewTestNamespace(func(ns string) {
		RegisterTestingT(t)
		Expect(kamel("install", "-n", ns, "--maven-repository", "https://my.repo.org/public/").Execute()).Should(BeNil())
		Eventually(configmap(ns, "camel-k-maven-settings")).Should(Not(BeNil()))
		Eventually(func()string {
			return configmap(ns, "camel-k-maven-settings")().Data["settings.xml"]
		}).Should(ContainSubstring("https://my.repo.org/public/"))
	})
}

