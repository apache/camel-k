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

package upgrade

import (
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	kerrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/operator-framework/api/pkg/lib/version"
	olm "github.com/operator-framework/api/pkg/operators/v1alpha1"
	corev1 "k8s.io/api/core/v1"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const catalogSourceName = "test-camel-k-source"

func TestOLMOperatorUpgrade(t *testing.T) {
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

	WithNewTestNamespace(t, func(g *WithT, ns string) {

		g.Expect(CreateOrUpdateCatalogSource(t, ns, catalogSourceName, prevIIB)).To(Succeed())
		ocp, err := openshift.IsOpenShift(TestClient(t))
		require.NoError(t, err)

		if ocp {
			// Wait for pull secret to be created in namespace
			// e.g. test-camel-k-source-dockercfg-zlltn
			secretPrefix := fmt.Sprintf("%s-dockercfg-", catalogSourceName)
			g.Eventually(SecretByName(t, ns, secretPrefix), TestTimeoutLong).Should(Not(BeNil()))
		}

		g.Eventually(CatalogSourcePodRunning(t, ns, catalogSourceName), TestTimeoutMedium).Should(BeNil())
		g.Eventually(CatalogSourcePhase(t, ns, catalogSourceName), TestTimeoutMedium).Should(Equal("READY"))

		// Set KAMEL_BIN only for this test - don't override the ENV variable for all tests
		g.Expect(os.Setenv("KAMEL_BIN", kamel)).To(Succeed())

		if len(CRDs(t)()) > 0 {
			// Clean up old installation - maybe leftover from another test
			if err := UninstallAll(t); err != nil && !kerrors.IsNotFound(err) {
				t.Error(err)
				t.FailNow()
			}
		}
		g.Eventually(CRDs(t)).Should(HaveLen(0))

		args := []string{
			"install",
			"-n", ns,
			"--olm=true",
			"--olm-source", catalogSourceName,
			"--olm-source-namespace", ns,
			"--base-image", defaults.BaseImage(),
		}

		if prevUpdateChannel != "" {
			args = append(args, "--olm-channel", prevUpdateChannel)
		}

		g.Expect(Kamel(t, args...).Execute()).To(Succeed())

		// Find the only one Camel K CSV
		noAdditionalConditions := func(csv olm.ClusterServiceVersion) bool {
			return true
		}
		g.Eventually(ClusterServiceVersionPhase(t, noAdditionalConditions, ns), TestTimeoutMedium).
			Should(Equal(olm.CSVPhaseSucceeded))

		// Refresh the test client to account for the newly installed CRDs
		RefreshClient(t)

		var prevCSVVersion version.OperatorVersion
		var newCSVVersion version.OperatorVersion

		// IntegrationPlatform should match at least on the version prefix
		// CSV patch version can be increased with the OperatorHub respin of the same Camel K release
		var prevIPVersionPrefix string
		var newIPVersionMajorMinorPatch string

		prevCSVVersion = ClusterServiceVersion(t, noAdditionalConditions, ns)().Spec.Version
		prevIPVersionPrefix = fmt.Sprintf("%d.%d", prevCSVVersion.Version.Major, prevCSVVersion.Version.Minor)
		t.Logf("Using Previous CSV Version: %s", prevCSVVersion.Version.String())

		// Check the operator pod is running
		g.Eventually(OperatorPodPhase(t, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		// Check the IntegrationPlatform has been reconciled
		g.Eventually(PlatformVersion(t, ns)).Should(ContainSubstring(prevIPVersionPrefix))

		name := "yaml"
		g.Expect(Kamel(t, "run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		kbindName := "timer-to-log"
		g.Expect(KamelBind(t, ns, "timer-source?message=Hello", "log-sink", "--name", kbindName).Execute()).To(Succeed())

		// Check the Integration runs correctly
		g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))
		if prevCSVVersion.Version.String() >= "2" { // since 2.0 Pipe, previously KameletBinding
			g.Eventually(PipeConditionStatus(t, ns, kbindName, v1.PipeConditionReady), TestTimeoutShort).
				Should(Equal(corev1.ConditionTrue))
		}
		g.Eventually(IntegrationPodPhase(t, ns, kbindName), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ns, kbindName, v1.IntegrationConditionReady), TestTimeoutLong).Should(Equal(corev1.ConditionTrue))

		// Check the Integration version matches that of the current operator
		g.Expect(IntegrationVersion(t, ns, name)()).To(ContainSubstring(prevIPVersionPrefix))
		g.Expect(IntegrationVersion(t, ns, kbindName)()).To(ContainSubstring(prevIPVersionPrefix))

		t.Run("OLM upgrade", func(t *testing.T) {
			// Trigger Camel K operator upgrade by updating the CatalogSource with the new index image
			g.Expect(CreateOrUpdateCatalogSource(t, ns, catalogSourceName, newIIB)).To(Succeed())

			if crossChannelUpgrade {
				t.Log("Patching Camel K OLM subscription channel.")
				subscription, err := GetSubscription(t, ns)
				g.Expect(err).To(BeNil())
				g.Expect(subscription).NotTo(BeNil())

				// Patch the Subscription to avoid conflicts with concurrent updates performed by OLM
				patch := fmt.Sprintf("{\"spec\":{\"channel\":%q}}", newUpdateChannel)
				g.Expect(
					TestClient(t).Patch(TestContext, subscription, ctrl.RawPatch(types.MergePatchType, []byte(patch))),
				).To(Succeed())
				// Assert the response back from the API server
				g.Expect(subscription.Spec.Channel).To(Equal(newUpdateChannel))
			}

			// The new CSV is installed
			g.Eventually(ClusterServiceVersionPhase(t, func(csv olm.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() != prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(Equal(olm.CSVPhaseSucceeded))

			// The old CSV is gone
			g.Eventually(ClusterServiceVersion(t, func(csv olm.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() == prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(BeNil())

			newCSVVersion = ClusterServiceVersion(t, noAdditionalConditions, ns)().Spec.Version
			newIPVersionMajorMinorPatch = fmt.Sprintf("%d.%d.%d", newCSVVersion.Version.Major, newCSVVersion.Version.Minor, newCSVVersion.Version.Patch)

			g.Expect(prevCSVVersion.Version.String()).NotTo(Equal(newCSVVersion.Version.String()))

			g.Eventually(OperatorPodPhase(t, ns), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			g.Eventually(OperatorImage(t, ns), TestTimeoutShort).Should(Equal(defaults.OperatorImage()))

			// Check the IntegrationPlatform has been reconciled
			g.Eventually(PlatformVersion(t, ns)).Should(ContainSubstring(newIPVersionMajorMinorPatch))
		})

		t.Run("Integration upgrade", func(t *testing.T) {
			// Clear the KAMEL_BIN environment variable so that the current version is used from now on
			g.Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

			// Check the Integration hasn't been upgraded
			g.Consistently(IntegrationVersion(t, ns, name), 5*time.Second, 1*time.Second).
				Should(ContainSubstring(prevIPVersionPrefix))

			// Rebuild the Integration
			g.Expect(Kamel(t, "rebuild", "--all", "-n", ns).Execute()).To(Succeed())
			if prevCSVVersion.Version.String() >= "2" {
				g.Eventually(PipeConditionStatus(t, ns, kbindName, v1.PipeConditionReady), TestTimeoutMedium).
					Should(Equal(corev1.ConditionTrue))
			}

			// Check the Integration runs correctly
			g.Eventually(IntegrationPodPhase(t, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))

			// Check the Integration version has been upgraded
			g.Eventually(IntegrationVersion(t, ns, name)).Should(ContainSubstring(newIPVersionMajorMinorPatch))
			g.Eventually(IntegrationVersion(t, ns, kbindName)).Should(ContainSubstring(newIPVersionMajorMinorPatch))

			// Check the previous kit is not garbage collected (skip Build - present in case of respin)
			prevCSVVersionMajorMinorPatch := fmt.Sprintf("%d.%d.%d",
				prevCSVVersion.Version.Major, prevCSVVersion.Version.Minor, prevCSVVersion.Version.Patch)
			g.Eventually(Kits(t, ns, KitWithVersion(prevCSVVersionMajorMinorPatch))).Should(HaveLen(2))
			// Check a new kit is created with the current version
			g.Eventually(Kits(t, ns, KitWithVersionPrefix(newIPVersionMajorMinorPatch))).Should(HaveLen(2))
			// Check the new kit is ready
			g.Eventually(Kits(t, ns, KitWithVersionPrefix(newIPVersionMajorMinorPatch), KitWithPhase(v1.IntegrationKitPhaseReady)),
				TestTimeoutMedium).Should(HaveLen(2))

			kit := Kits(t, ns, KitWithVersionPrefix(newIPVersionMajorMinorPatch), KitWithLabels(map[string]string{"camel.apache.org/created.by.name": name}))()[0]
			kitKbind := Kits(t, ns, KitWithVersionPrefix(newIPVersionMajorMinorPatch), KitWithLabels(map[string]string{"camel.apache.org/created.by.name": kbindName}))()[0]

			// Check the Integration uses the new kit
			g.Eventually(IntegrationKit(t, ns, name), TestTimeoutMedium).Should(Equal(kit.Name))
			g.Eventually(IntegrationKit(t, ns, kbindName), TestTimeoutMedium).Should(Equal(kitKbind.Name))
			// Check the Integration Pod uses the new image
			g.Eventually(IntegrationPodImage(t, ns, name)).Should(Equal(kit.Status.Image))
			g.Eventually(IntegrationPodImage(t, ns, kbindName)).Should(Equal(kitKbind.Status.Image))

			// Check the Integration runs correctly
			g.Eventually(IntegrationPodPhase(t, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodPhase(t, ns, kbindName)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationConditionStatus(t, ns, kbindName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Clean up
			g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
			// Delete Integration Platform as it does not get removed with uninstall and might cause next tests to fail
			DeletePlatform(t, ns)()
			g.Expect(Kamel(t, "uninstall", "-n", ns).Execute()).To(Succeed())
			// Clean up cluster-wide resources that are not removed by OLM
			g.Expect(Kamel(t, "uninstall", "--all", "-n", ns, "--olm=false").Execute()).To(Succeed())
		})
	})
}
