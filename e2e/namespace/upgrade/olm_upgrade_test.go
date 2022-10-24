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
	"time"

	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/api/pkg/lib/version"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/openshift"
)

const catalogSourceName = "test-camel-k-source"

func TestOLMAutomaticUpgrade(t *testing.T) {
	prevIIB := os.Getenv("CAMEL_K_PREV_IIB")
	newIIB := os.Getenv("CAMEL_K_NEW_IIB")
	kamel := os.Getenv("RELEASED_KAMEL_BIN")

	// optional options
	prevUpdateChannel := os.Getenv("CAMEL_K_PREV_UPGRADE_CHANNEL")
	newUpdateChannel := os.Getenv("CAMEL_K_NEW_UPGRADE_CHANNEL")

	if prevIIB == "" || newIIB == "" {
		t.Skip("OLM Upgrade test requires the CAMEL_K_PREV_IIB and CAMEL_K_NEW_IIB environment variables")
	}

	crossChannelUpgrade := false
	if prevUpdateChannel != "" && newUpdateChannel != "" && prevUpdateChannel != newUpdateChannel {
		crossChannelUpgrade = true
		t.Logf("Testing cross-OLM channel upgrade %s -> %s", prevUpdateChannel, newUpdateChannel)
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(createOrUpdateCatalogSource(ns, catalogSourceName, prevIIB)).To(Succeed())
		ocp, err := openshift.IsOpenShift(TestClient())
		assert.Nil(t, err)

		if ocp {
			// Wait for pull secret to be created in namespace
			// eg. test-camel-k-source-dockercfg-zlltn
			secretPrefix := fmt.Sprintf("%s-dockercfg-", catalogSourceName)
			Eventually(SecretByName(ns, secretPrefix), TestTimeoutLong).Should(Not(BeNil()))
		}

		Eventually(catalogSourcePodRunning(ns, catalogSourceName), TestTimeoutMedium).Should(BeNil())
		Eventually(catalogSourcePhase(ns, catalogSourceName), TestTimeoutMedium).Should(Equal("READY"))

		// Set KAMEL_BIN only for this test - don't override the ENV variable for all tests
		Expect(os.Setenv("KAMEL_BIN", kamel)).To(Succeed())

		args := []string{
			"install",
			"-n", ns,
			"--olm=true",
			"--olm-source", catalogSourceName,
			"--olm-source-namespace", ns,
		}

		if prevUpdateChannel != "" {
			args = append(args, "--olm-channel", prevUpdateChannel)
		}

		Expect(Kamel(args...).Execute()).To(Succeed())

		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		Eventually(clusterServiceVersionPhase(noAdditionalConditions, ns), TestTimeoutMedium).
			Should(Equal(olm.CSVPhaseSucceeded))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		var prevCSVVersion version.OperatorVersion
		var newCSVVersion version.OperatorVersion

		// IntegrationPlatform should match at least on the version prefix
		// CSV patch version can be increased with the OperatorHub respin of the same Camel K release
		var prevIPVersionPrefix string
		var newIPVersionPrefix string

		prevCSVVersion = clusterServiceVersion(noAdditionalConditions, ns)().Spec.Version
		prevIPVersionPrefix = fmt.Sprintf("%d.%d", prevCSVVersion.Version.Major, prevCSVVersion.Version.Minor)
		t.Logf("Using Previous CSV Version: %s", prevCSVVersion.Version.String())

		// Check the operator pod is running
		Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Check the IntegrationPlatform has been reconciled
		Eventually(PlatformVersion(ns)).Should(ContainSubstring(prevIPVersionPrefix))

		name := "yaml"
		Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		// Check the Integration runs correctly
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))

		// Check the Integration version matches that of the current operator
		Expect(IntegrationVersion(ns, name)()).To(ContainSubstring(prevIPVersionPrefix))

		t.Run("OLM upgrade", func(t *testing.T) {
			// Trigger Camel K operator upgrade by updating the CatalogSource with the new index image
			Expect(createOrUpdateCatalogSource(ns, catalogSourceName, newIIB)).To(Succeed())

			if crossChannelUpgrade {
				t.Log("Patching Camel K OLM subscription channel.")
				subscription, err := getSubscription(ns)
				Expect(err).To(BeNil())
				Expect(subscription).NotTo(BeNil())

				// Patch the Subscription to avoid conflicts with concurrent updates performed by OLM
				patch := fmt.Sprintf("{\"spec\":{\"channel\":%q}}", newUpdateChannel)
				Expect(
					TestClient().Patch(TestContext, subscription, ctrl.RawPatch(types.MergePatchType, []byte(patch))),
				).To(Succeed())
				// Assert the response back from the API server
				Expect(subscription.Spec.Channel).To(Equal(newUpdateChannel))
			}

			// Check the previous CSV is being replaced
			Eventually(clusterServiceVersionPhase(func(csv olm.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() == prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(Equal(olm.CSVPhaseReplacing))

			// The new CSV is installed
			Eventually(clusterServiceVersionPhase(func(csv olm.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() != prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(Equal(olm.CSVPhaseSucceeded))

			// The old CSV is gone
			Eventually(clusterServiceVersion(func(csv olm.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() == prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(BeNil())

			newCSVVersion = clusterServiceVersion(noAdditionalConditions, ns)().Spec.Version
			newIPVersionPrefix = fmt.Sprintf("%d.%d", newCSVVersion.Version.Major, newCSVVersion.Version.Minor)

			Expect(prevCSVVersion.Version.String()).NotTo(Equal(newCSVVersion.Version.String()))

			Eventually(OperatorPodPhase(ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(OperatorImage(ns), TestTimeoutShort).Should(Equal(defaults.OperatorImage()))

			// Check the IntegrationPlatform has been reconciled
			Eventually(PlatformVersion(ns)).Should(ContainSubstring(newIPVersionPrefix))
		})

		t.Run("Integration upgrade", func(t *testing.T) {
			// Clear the KAMEL_BIN environment variable so that the current version is used from now on
			Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

			// Check the Integration hasn't been upgraded
			Consistently(IntegrationVersion(ns, name), 5*time.Second, 1*time.Second).
				Should(ContainSubstring(prevIPVersionPrefix))

			// Rebuild the Integration
			Expect(Kamel("rebuild", name, "-n", ns).Execute()).To(Succeed())

			// Check the Integration runs correctly
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))

			// Check the Integration version has been upgraded
			Eventually(IntegrationVersion(ns, name)).Should(ContainSubstring(newIPVersionPrefix))

			// Check the previous kit is not garbage collected
			Eventually(Kits(ns, KitWithVersion(prevCSVVersion.String()))).Should(HaveLen(1))
			// Check a new kit is created with the current version
			Eventually(Kits(ns, KitWithVersion(defaults.Version)),
				TestTimeoutMedium).Should(HaveLen(1))
			// Check the new kit is ready
			Eventually(Kits(ns, KitWithVersion(defaults.Version), KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutMedium).Should(HaveLen(1))

			kit := Kits(ns, KitWithVersion(defaults.Version))()[0]

			// Check the Integration uses the new kit
			Eventually(IntegrationKit(ns, name), TestTimeoutMedium).Should(Equal(kit.Name))
			// Check the Integration Pod uses the new image
			Eventually(IntegrationPodImage(ns, name)).Should(Equal(kit.Status.Image))

			// Check the Integration runs correctly
			Eventually(IntegrationPodPhase(ns, name)).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Clean up
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Expect(Kamel("uninstall", "-n", ns).Execute()).To(Succeed())
			// Clean up cluster-wide resources that are not removed by OLM
			Expect(Kamel("uninstall", "--all", "-n", ns, "--olm=false").Execute()).To(Succeed())
		})
	})
}
