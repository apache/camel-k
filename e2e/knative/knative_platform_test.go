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

package knative

import (
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	"github.com/stretchr/testify/assert"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/dsl"
	"github.com/apache/camel-k/pkg/util/knative"
)

func TestKnativePlatform(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installed, err := knative.IsInstalled(TestClient())
		Expect(err).NotTo(HaveOccurred())
		if !installed {
			t.Error("Knative not installed in the cluster")
			t.FailNow()
		}

		operatorID := "camel-k-knative"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		Eventually(PlatformProfile(ns), TestTimeoutShort).Should(Equal(v1.TraitProfile("")))
		cluster := Platform(ns)().Status.Cluster

		t.Run("run yaml on cluster profile", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/yaml.yaml", "--profile", string(cluster)).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationProfile(ns, "yaml"), TestTimeoutShort).Should(Equal(v1.TraitProfile(string(cluster))))
			// Change something in the integration to produce a redeploy
			Expect(UpdateIntegration(ns, "yaml", func(it *v1.Integration) {
				it.Spec.Profile = ""
				content, err := dsl.ToYamlDSL(it.Spec.Flows)
				assert.NoError(t, err)
				newData := strings.ReplaceAll(string(content), "string!", "string!!!")
				newFlows, err := dsl.FromYamlDSLString(newData)
				assert.NoError(t, err)
				it.Spec.Flows = newFlows
			})).To(Succeed())
			// Spec profile should be reset by "kamel run"
			Eventually(IntegrationSpecProfile(ns, "yaml")).Should(Equal(v1.TraitProfile("")))
			// When integration is running again ...
			Eventually(IntegrationPhase(ns, "yaml")).Should(Equal(v1.IntegrationPhaseRunning))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!!!"))
			// It should keep the old profile saved in status
			Eventually(IntegrationProfile(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.TraitProfile(cluster)))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run yaml on automatic profile", func(t *testing.T) {
			Expect(KamelRunWithID(operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationProfile(ns, "yaml"), TestTimeoutShort).Should(Equal(v1.TraitProfileKnative))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
