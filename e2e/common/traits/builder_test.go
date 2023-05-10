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

package traits

import (
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestBuilderTrait(t *testing.T) {
	RegisterTestingT(t)

	name := "java"

	t.Run("Run build strategy routine", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.strategy=routine").Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		integrationKitName := IntegrationKit(ns, name)()
		builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)
		Eventually(BuildConfig(ns, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyRoutine))
		// Default resource CPU Check
		Eventually(BuildConfig(ns, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal(""))
		Eventually(BuildConfig(ns, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal(""))
		Eventually(BuildConfig(ns, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal(""))
		Eventually(BuildConfig(ns, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal(""))

		Eventually(BuilderPod(ns, builderKitName), TestTimeoutShort).Should(BeNil())

		// We need to remove the kit as well
		Expect(Kamel("reset", "-n", ns).Execute()).To(Succeed())
	})

	t.Run("Run build resources configuration", func(t *testing.T) {
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "builder.request-cpu=500m",
			"-t", "builder.limit-cpu=1000m",
			"-t", "builder.request-memory=2Gi",
			"-t", "builder.limit-memory=3Gi",
		).Execute()).To(Succeed())

		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		integrationKitName := IntegrationKit(ns, name)()
		builderKitName := fmt.Sprintf("camel-k-%s-builder", integrationKitName)

		Eventually(BuildConfig(ns, integrationKitName)().Strategy, TestTimeoutShort).Should(Equal(v1.BuildStrategyPod))
		Eventually(BuildConfig(ns, integrationKitName)().RequestCPU, TestTimeoutShort).Should(Equal("500m"))
		Eventually(BuildConfig(ns, integrationKitName)().LimitCPU, TestTimeoutShort).Should(Equal("1000m"))
		Eventually(BuildConfig(ns, integrationKitName)().RequestMemory, TestTimeoutShort).Should(Equal("2Gi"))
		Eventually(BuildConfig(ns, integrationKitName)().LimitMemory, TestTimeoutShort).Should(Equal("3Gi"))

		Eventually(BuilderPod(ns, builderKitName), TestTimeoutShort).ShouldNot(BeNil())
		// Let's assert we set the resources on the builder container
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Name, TestTimeoutShort).Should(Equal("builder"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Requests.Cpu().String(), TestTimeoutShort).Should(Equal("500m"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Limits.Cpu().String(), TestTimeoutShort).Should(Equal("1"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Requests.Memory().String(), TestTimeoutShort).Should(Equal("2Gi"))
		Eventually(BuilderPod(ns, builderKitName)().Spec.InitContainers[0].Resources.Limits.Memory().String(), TestTimeoutShort).Should(Equal("3Gi"))

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
