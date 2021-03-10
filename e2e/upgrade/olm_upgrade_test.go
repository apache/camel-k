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
	. "github.com/apache/camel-k/e2e/support"
	. "github.com/onsi/gomega"
	"github.com/operator-framework/api/pkg/lib/version"
	"github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"os"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
	"testing"
)

const CATALOG_SOURCE_NAME = "test-camel-k-source"

func TestOLMAutomaticUpgrade(t *testing.T) {
	prevIIB := os.Getenv("CAMEL_K_PREV_IIB")
	newIIB := os.Getenv("CAMEL_K_NEW_IIB")
	kamel := os.Getenv("RELEASED_KAMEL_BIN")

	if prevIIB == "" || newIIB == "" {
		t.Skip("OLM Upgrade test needs CAMEL_K_PREV_IIB and CAMEL_K_PREV_IIB ENV variables")
	}

	WithNewTestNamespace(t, func(ns string) {
		Expect(createCatalogSource(CATALOG_SOURCE_NAME, prevIIB, ns)).To(Succeed())
		Eventually(CatalogSourcePhase(CATALOG_SOURCE_NAME, ns), TestTimeoutMedium).Should(Equal("READY"))

		//set KAMEL_BIN only for this test - don't override the ENV variable for all tests
		Expect(os.Setenv("KAMEL_BIN", kamel)).To(Succeed())

		Expect(Kamel("install", "-n", ns, "--olm=true", "--olm-source", CATALOG_SOURCE_NAME, "--olm-source-namespace", ns).Execute()).To(Succeed())

		//find the only one Camel-K CSV
		noAdditionalConditions := func(csv v1alpha1.ClusterServiceVersion) bool {
			return true
		}
		Eventually(CKClusterServiceVersionPhase(noAdditionalConditions, ns), TestTimeoutMedium).Should(Equal(v1alpha1.CSVPhaseSucceeded))

		// Refresh the test client to account for the newly installed CRDs
		SyncClient()

		var prevCSVVersion version.OperatorVersion
		var newCSVVersion version.OperatorVersion

		// IntegrationPlatform should match at least on the version prefix
		// CSV patch version can be increased with the OperatorHub respin of the same Camel-K release
		var prevIPVersionPrefix string
		var newIPVersionPrefix string

		prevCSVVersion = CKClusterServiceVersion(noAdditionalConditions, ns)().Spec.Version
		prevIPVersionPrefix = fmt.Sprintf("%d.%d", prevCSVVersion.Version.Major, prevCSVVersion.Version.Minor)

		Expect(OperatorPod(ns)).ToNot(BeNil())

		Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.PodRunning))

		Eventually(PlatformVersion(ns)).Should(ContainSubstring(prevIPVersionPrefix))
		Expect(IntegrationVersion(ns, "yaml")()).To(ContainSubstring(prevIPVersionPrefix))

		t.Run("OLM upgrade", func(t *testing.T) {

			//invoke OLM upgrade
			Expect(createCatalogSource(CATALOG_SOURCE_NAME, newIIB, ns)).To(Succeed())

			// previous CSV is REPLACING
			Eventually(CKClusterServiceVersionPhase(func(csv v1alpha1.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() == prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(Equal(v1alpha1.CSVPhaseReplacing))

			// the new version is installed
			Eventually(CKClusterServiceVersionPhase(func(csv v1alpha1.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() != prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(Equal(v1alpha1.CSVPhaseSucceeded))

			// the old version is gone
			Eventually(CKClusterServiceVersion(func(csv v1alpha1.ClusterServiceVersion) bool {
				return csv.Spec.Version.Version.String() == prevCSVVersion.Version.String()
			}, ns), TestTimeoutMedium).Should(BeNil())

			newCSVVersion = CKClusterServiceVersion(noAdditionalConditions, ns)().Spec.Version
			newIPVersionPrefix = fmt.Sprintf("%d.%d", newCSVVersion.Version.Major, newCSVVersion.Version.Minor)

			Expect(prevCSVVersion.Version.String()).NotTo(Equal(newCSVVersion.Version.String()))

			Eventually(OperatorPodPhase(ns)).Should(Equal(v1.PodRunning))

			Eventually(PlatformVersion(ns)).Should(ContainSubstring(newIPVersionPrefix))
		})

		t.Run("Integration upgrade", func(t *testing.T) {

			// Clear the KAMEL_BIN environment variable so that the current version is used from now on
			Expect(os.Setenv("KAMEL_BIN", "")).To(Succeed())

			Expect(IntegrationVersion(ns, "yaml")()).To(ContainSubstring(prevIPVersionPrefix))
			Expect(Kamel("rebuild", "yaml", "-n", ns).Execute()).To(Succeed())

			Eventually(IntegrationVersion(ns, "yaml"), TestTimeoutMedium).Should(ContainSubstring(newIPVersionPrefix))
			Eventually(IntegrationPodPhase(ns, "yaml")).Should(Equal(v1.PodRunning))

			// Clean up
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
			Expect(Kamel("uninstall", "-n", ns).Execute()).To(Succeed())
		})

	})
}

func createCatalogSource(name string, image string, ns string) error {
	catalogSource := v1alpha1.CatalogSource{
		TypeMeta: metav1.TypeMeta{
			Kind:       "CatalogSource",
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      name,
		},
	}
	key := ctrl.ObjectKey{
		Namespace: ns,
		Name:      name,
	}
	if err := TestClient().Get(TestContext, key, &catalogSource); errors.IsNotFound(err) {
		catalogSource.Spec = v1alpha1.CatalogSourceSpec{
			Image:       image,
			SourceType:  "grpc",
			DisplayName: "OLM upgrade test Catalog",
			Publisher:   "grpc",
		}
		return TestClient().Create(TestContext, &catalogSource)
	} else {
		catalogSource.Spec.Image = image
		return TestClient().Update(TestContext, &catalogSource)
	}

}
