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

package common

import (
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
)

func TestBasicInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	})
}

func TestAlternativeImageInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--operator-image", "x/y:latest").Execute()).Should(BeNil())
		Eventually(OperatorImage(ns)).Should(Equal("x/y:latest"))
	})
}

func TestKitMainInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Expect(Kamel("kit", "create", "timer", "-d", "camel:timer", "-n", ns).Execute()).Should(BeNil())
		Eventually(Build(ns, "timer")).ShouldNot(BeNil())
	})
}

func TestMavenRepositoryInstallation(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--maven-repository", "https://my.repo.org/public/").Execute()).Should(BeNil())
		Eventually(Configmap(ns, "camel-k-maven-settings")).Should(Not(BeNil()))
		Eventually(func() string {
			return Configmap(ns, "camel-k-maven-settings")().Data["settings.xml"]
		}).Should(ContainSubstring("https://my.repo.org/public/"))
	})
}
