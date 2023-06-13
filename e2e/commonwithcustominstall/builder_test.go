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

package commonwithcustominstall

import (
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuilderTimeout(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		operatorID := fmt.Sprintf("camel-k-%s", ns)
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())
		Eventually(OperatorPod(ns)).ShouldNot(BeNil())
		Eventually(Platform(ns)).ShouldNot(BeNil())
		Eventually(PlatformConditionStatus(ns, v1.IntegrationPlatformConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))

		pl := Platform(ns)()
		// set a short timeout to simulate the build timeout
		pl.Spec.Build.Timeout = &metav1.Duration{
			Duration: 10 * time.Second,
		}
		TestClient().Update(TestContext, pl)
		Eventually(Platform(ns)).ShouldNot(BeNil())
		Eventually(PlatformTimeout(ns)).Should(Equal(
			&metav1.Duration{
				Duration: 10 * time.Second,
			},
		))

		t.Run("run yaml", func(t *testing.T) {
			name := "yaml"
			Expect(KamelRunWithID(operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			// As the build hits timeout, it keeps trying building
			Eventually(IntegrationPhase(ns, name)).Should(Equal(v1.IntegrationPhaseBuildingKit))
			integrationKitName := IntegrationKit(ns, name)()
			builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
			Eventually(BuilderPodPhase(ns, builderKitName)).Should(Equal(corev1.PodPending))
			Eventually(BuildPhase(ns, integrationKitName)).Should(Equal(v1.BuildPhaseRunning))
			// After a few minutes (5 max retries), this has to be in error state
			Eventually(BuildPhase(ns, integrationKitName), TestTimeoutMedium).Should(Equal(v1.BuildPhaseError))
			Eventually(IntegrationPhase(ns, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseError))
			Eventually(BuildFailureRecovery(ns, integrationKitName), TestTimeoutMedium).Should(Equal(5))
			Eventually(BuilderPodPhase(ns, builderKitName), TestTimeoutMedium).Should(Equal(corev1.PodFailed))
		})
	})
}
