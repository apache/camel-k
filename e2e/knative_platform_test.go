// +build knative

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
	"strings"
	"testing"

	. "github.com/apache/camel-k/e2e/support"
	"github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/knative"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestKnativePlatformTest(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		if !knative.IsEnabledInNamespace(TestContext, TestClient, ns) {
			t.Error("Knative not installed in the cluster")
			t.FailNow()
		}

		Expect(Kamel("install", "-n", ns).Execute()).Should(BeNil())
		Eventually(PlatformPhase(ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		Eventually(PlatformProfile(ns), TestTimeoutShort).Should(Equal(v1.TraitProfile("")))
		cluster := Platform(ns)().Status.Cluster

		t.Run("run yaml on cluster profile", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml", "--profile", string(cluster)).Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationProfile(ns, "yaml"), TestTimeoutShort).Should(Equal(v1.TraitProfile(string(cluster))))
			// Change something in the integration to produce a redeploy
			Expect(UpdateIntegration(ns, "yaml", func(it *v1.Integration) {
				it.Spec.Profile = v1.TraitProfile("")
				it.Spec.Sources[0].Content = strings.ReplaceAll(it.Spec.Sources[0].Content, "string!", "string!!!")
			})).To(BeNil())
			// Spec profile should be reset by "kamel run"
			Eventually(IntegrationSpecProfile(ns, "yaml")).Should(Equal(v1.TraitProfile("")))
			// When integration is running again ...
			Eventually(IntegrationPhase(ns, "yaml")).Should(Equal(v1.IntegrationPhaseRunning))
			Eventually(IntegrationLogs(ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!!!"))
			// It should keep the old profile saved in status
			Eventually(IntegrationProfile(ns, "yaml"), TestTimeoutMedium).Should(Equal(v1.TraitProfile(string(cluster))))

			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("run yaml on automatic profile", func(t *testing.T) {
			RegisterTestingT(t)
			Expect(Kamel("run", "-n", ns, "files/yaml.yaml").Execute()).Should(BeNil())
			Eventually(IntegrationPodPhase(ns, "yaml"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationProfile(ns, "yaml"), TestTimeoutShort).Should(Equal(v1.TraitProfileKnative))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

	})
}
