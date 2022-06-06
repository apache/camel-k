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

package common

import (
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestPlatformlessRun(t *testing.T) {
	needsExternalRepo := os.Getenv("STAGING_RUNTIME_REPO") != "" || os.Getenv("KAMEL_INSTALL_MAVEN_REPOSITORIES") != ""
	ocp, err := openshift.IsOpenShift(TestClient())
	assert.Nil(t, err)
	if needsExternalRepo || !ocp {
		t.Skip("This test is for OpenShift only and cannot work when a custom platform configuration is needed")
		return
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(KamelInstall(ns).Execute()).To(Succeed())

		// Delete the platform from the namespace before running the integration
		Eventually(DeletePlatform(ns)).Should(BeTrue())

		Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		// Platform should be recreated
		Eventually(Platform(ns)).ShouldNot(BeNil())
		Eventually(PlatformProfile(ns)).Should(Equal(v1.TraitProfile("")))
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
