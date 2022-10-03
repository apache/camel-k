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

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

var sampleJar = "https://raw.githubusercontent.com/apache/camel-k/main/e2e/global/common/traits/files/jvm/sample-1.0.jar"

func operatorID(ns string) string {
	return fmt.Sprintf("camel-k-%s", ns)
}

func installWithID(ns string) {
	Expect(KamelInstallWithID(operatorID(ns), ns).Execute()).To(Succeed())
	Eventually(OperatorPod(ns)).ShouldNot(BeNil())
	Eventually(PlatformPhase(ns), TestTimeoutLong).Should(Equal(v1.IntegrationPlatformPhaseReady))
}

func TestKamelCLIRunGitHubExampleJava(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns,
			"github:apache/camel-k/e2e/namespace/install/files/Java.java").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
	})
}

func TestKamelCLIRunGitHubExampleJavaRaw(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns,
			"https://raw.githubusercontent.com/apache/camel-k/main/e2e/namespace/install/files/Java.java").Execute()).
			To(Succeed())
		Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
	})
}

func TestKamelCLIRunGitHubExampleJavaBranch(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns,
			"github:apache/camel-k/e2e/namespace/install/files/Java.java?branch=main").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
	})
}

func TestKamelCLIRunGitHubExampleGistID(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		name := "github-gist-id"
		Expect(KamelRunWithID(operatorID(ns), ns, "--name", name,
			"gist:e2c3f9a5fd0d9e79b21b04809786f17a").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Tick!"))
	})
}

func TestKamelCLIRunGitHubExampleGistURL(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		name := "github-gist-url"
		Expect(KamelRunWithID(operatorID(ns), ns, "--name", name,
			"https://gist.github.com/lburgazzoli/e2c3f9a5fd0d9e79b21b04809786f17a").Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Tick!"))
	})
}

func TestKamelCLIRunAndUpdate(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		name := "run"
		Expect(KamelRunWithID(operatorID(ns), ns, "files/run.yaml", "--name", name).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magic default"))

		// Re-run the Integration with an updated configuration
		Expect(KamelRunWithID(operatorID(ns), ns, "files/run.yaml", "--name", name, "-p", "property=value").Execute()).
			To(Succeed())

		// Check the Deployment has progressed successfully
		Eventually(DeploymentCondition(ns, name, appsv1.DeploymentProgressing), TestTimeoutShort).
			Should(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal("NewReplicaSetAvailable"),
			}))

		// Check the new configuration is taken into account
		Eventually(IntegrationPodPhase(ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).
			Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, name), TestTimeoutShort).Should(ContainSubstring("Magic value"))

	})
}

/*
 * TODO
 * The dependency cannot be read by maven while building. See #3708
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestKamelCLIRunWithHttpDependency(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		fmt.Println("OperatorID: ", operatorID(ns))
		Expect(KamelRunWithID(operatorID(ns), ns, "../../../global/common/traits/files/jvm/Classpath.java",
			"-d", sampleJar, "--verbose",
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "classpath"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "classpath", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "classpath"), TestTimeoutShort).Should(ContainSubstring("Hello World!"))
	})
}

/*
 * TODO
 * The dependency cannot be read by maven while building. See #3708
 *
 * Adding CAMEL_K_TEST_SKIP_PROBLEMATIC env var for the moment.
 */
func TestKamelCLIRunHttpDependencyUsingOptions(t *testing.T) {
	if os.Getenv("CAMEL_K_TEST_SKIP_PROBLEMATIC") == "true" {
		t.Skip("WARNING: Test marked as problematic ... skipping")
	}
	WithNewTestNamespace(t, func(ns string) {
		installWithID(ns)

		Expect(KamelRunWithID(operatorID(ns), ns, "../../../global/common/traits/files/jvm/Classpath.java",
			"-d", sampleJar, "--verbose",
			"-d", "https://raw.githubusercontent.com/apache/camel-k/main/e2e/namespace/install/cli/files/Java.java|targetPath=/tmp/foo",
		).Execute()).To(Succeed())
		Eventually(IntegrationPodPhase(ns, "classpath"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		Eventually(IntegrationConditionStatus(ns, "classpath", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationLogs(ns, "classpath"), TestTimeoutShort).Should(ContainSubstring("Hello World!"))
	})
}
