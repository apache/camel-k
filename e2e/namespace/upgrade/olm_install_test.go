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
	"fmt"
	"os"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/openshift"
)

const installCatalogSourceName = "test-camel-k-source"

func TestOLMInstallation(t *testing.T) {
	// keep option names compatible with the upgrade test
	newIIB := os.Getenv("CAMEL_K_NEW_IIB")

	// optional options
	newUpdateChannel := os.Getenv("CAMEL_K_NEW_UPGRADE_CHANNEL")

	if newIIB == "" {
		t.Skip("OLM fresh install test requires the CAMEL_K_NEW_IIB environment variable")
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(createOrUpdateCatalogSource(ns, installCatalogSourceName, newIIB)).To(Succeed())

		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)

		if ocp {
			// Wait for pull secret to be created in namespace
			// eg. test-camel-k-source-dockercfg-zlltn
			secretPrefix := fmt.Sprintf("%s-dockercfg-", installCatalogSourceName)
			Eventually(SecretByName(ns, secretPrefix), TestTimeoutLong).Should(Not(BeNil()))
		}

		Eventually(catalogSourcePodRunning(ns, installCatalogSourceName), TestTimeoutMedium).Should(BeNil())
		Eventually(catalogSourcePhase(ns, installCatalogSourceName), TestTimeoutLong).Should(Equal("READY"))

		args := []string{"install", "-n", ns, "--olm=true", "--olm-source", installCatalogSourceName, "--olm-source-namespace", ns}

		if newUpdateChannel != "" {
			args = append(args, "--olm-channel", newUpdateChannel)
		}

		Expect(Kamel(args...).Execute()).To(Succeed())

		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		Eventually(clusterServiceVersionPhase(noAdditionalConditions, ns), TestTimeoutMedium).Should(Equal(olm.CSVPhaseSucceeded))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		csvVersion := clusterServiceVersion(noAdditionalConditions, ns)().Spec.Version
		ipVersionPrefix := fmt.Sprintf("%d.%d", csvVersion.Version.Major, csvVersion.Version.Minor)
		t.Logf("CSV Version installed: %s", csvVersion.Version.String())

		// Check the operator pod is running
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(OperatorImage(ns), TestTimeoutShort).Should(Equal(defaults.OperatorImage()))

		// Check the IntegrationPlatform has been reconciled
		Eventually(PlatformVersion(ns)).Should(ContainSubstring(ipVersionPrefix))

		name := "yaml"
		Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		// Check the Integration runs correctly
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))

		// Check the Integration version matches that of the current operator
		Expect(IntegrationVersion(ns, name)()).To(ContainSubstring(ipVersionPrefix))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		Expect(Kamel("uninstall", "-n", ns).Execute()).To(Succeed())
		// Clean up cluster-wide resources that are not removed by OLM
		Expect(Kamel("uninstall", "--all", "-n", ns, "--olm=false").Execute()).To(Succeed())
	})
}
