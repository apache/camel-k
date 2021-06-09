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

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	"github.com/stretchr/testify/assert"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/openshift"
)

func TestRunExamplesFromGitHub(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		t.Run("run java from GitHub", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "github:apache/camel-k/e2e/common/files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run java from GitHub (RAW)", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "https://raw.githubusercontent.com/apache/camel-k/main/e2e/common/files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationCondition(ns, "java", camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run from GitHub Gist (ID)", func(t *testing.T) {
			name := "github-gist-id"
			Expect(Kamel("run", "-n", ns, "--name", name, "gist:e2c3f9a5fd0d9e79b21b04809786f17a").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Tick!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("run from GitHub Gist (URL)", func(t *testing.T) {
			name := "github-gist-url"
			Expect(Kamel("run", "-n", ns, "--name", name, "https://gist.github.com/lburgazzoli/e2c3f9a5fd0d9e79b21b04809786f17a").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Tick!"))
			Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func TestRunAndUpdate(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())

		name := "run"
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magic default"))

		// Re-run the Integration with an updated configuration
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name, "-p", "property=value").Execute()).To(Succeed())

		// Check the Deployment has progressed successfully
		Eventually(DeploymentCondition(ns, name, appsv1.DeploymentProgressing), TestTimeoutShort).Should(MatchFields(IgnoreExtras, Fields{
			"Status": Equal(corev1.ConditionTrue),
			"Reason": Equal("NewReplicaSetAvailable"),
		}))

		// Check the new configuration is taken into account
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magic value"))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

//
// Try and run integration specfying no registry on non-openshift cluster
// Should leave integration in Error phase and the integration kit in Cannot Build phase
//
func TestRunNoRegistryCannotBuild(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		Expect(Kamel("install", "-n", ns, "--registry", "none").Execute()).To(Succeed())
		Eventually(Platform(ns)).ShouldNot(BeNil())

		if ocp, err := openshift.IsOpenShift(TestClient()); ocp {
			assert.Nil(t, err)
			t.Skip("Test not applicable since Openshift always has a registry available.")
			return
		}

		name := "run"
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(camelv1.IntegrationPhaseError))
		Eventually(IntegrationKitPhase(ns, name), TestTimeoutShort).Should(Equal(camelv1.IntegrationKitPhaseCannotBuild))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

//
// Try and run already-built integration specfying no registry on non-openshift cluster
//
func TestRunNoRegistryRunBuiltIntegration(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		//
		// Install kamel with registry and create an integration
		//
		Expect(Kamel("install", "-n", ns).Execute()).To(Succeed())
		Eventually(Platform(ns)).ShouldNot(BeNil())

		if ocp, err := openshift.IsOpenShift(TestClient()); ocp {
			assert.Nil(t, err)
			t.Skip("Test not applicable since Openshift always has a registry available.")
			return
		}

		//
		// Creates an integration and ensures its ready
		// This will build the dependent integration kit
		//
		name := "run"
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		//
		// Get the name of the integration kit
		//
		itk := IntegrationKit(ns, name)()
		assert.NotEmpty(t, itk)

		//
		// Reinstall disabling the registry in the platform
		//
		Expect(Kamel("install", "-n", ns, "--force", "true", "--registry", "none").Execute()).To(Succeed())
		Eventually(func() string {
			return Platform(ns)().Spec.Build.Registry.Address
		}).Should(Equal(camelv1.IntegrationPlatformRegistryDisabled))

		spec := Platform(ns)().Spec
		assert.Equal(t, spec.Build.PublishStrategy, camelv1.IntegrationPlatformBuildPublishStrategyDisabled)
		assert.Equal(t, spec.Build.BuildStrategy, camelv1.IntegrationPlatformBuildStrategyDisabled)

		//
		// Platform now disabled due to --registry=none
		// Run the same integration specifying the existing built kit
		// Integration kit does not need any building so integration should just run
		//
		name2 := "run2"
		Expect(Kamel("run", "-n", ns, "files/run.yaml", "--name", name2, "--kit", itk).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name2), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationCondition(ns, name2, camelv1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))

		// Clean up
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
