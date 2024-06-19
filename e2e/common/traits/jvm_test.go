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
	"context"
	"os"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestJVMTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Store a configmap holding a jar
		var cmData = make(map[string][]byte)
		// We calculate the expected content
		source, err := os.ReadFile("./files/jvm/sample-1.0.jar")
		require.NoError(t, err)
		cmData["sample-1.0.jar"] = source
		err = CreateBinaryConfigmap(t, ctx, ns, "my-deps", cmData)
		require.NoError(t, err)

		t.Run("JVM trait classpath", func(t *testing.T) {
			name := RandomizedSuffixName("classpath")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/jvm/Classpath.java",
				"--name", name,
				"--resource", "configmap:my-deps",
				"-t", "jvm.classpath=/etc/camel/resources.d/_configmaps/my-deps/sample-1.0.jar").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello World!"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, name)()
			jvmTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "jvm")
			g.Expect(jvmTrait).ToNot(BeNil())
			g.Expect(len(jvmTrait)).To(Equal(1))
			g.Expect(jvmTrait["classpath"]).To(Equal("/etc/camel/resources.d/_configmaps/my-deps/sample-1.0.jar"))
			mountTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "mount")
			g.Expect(mountTrait).ToNot(BeNil())
			g.Expect(len(mountTrait)).To(Equal(1))

		})

		t.Run("JVM trait classpath on deprecated path", func(t *testing.T) {
			name := RandomizedSuffixName("classpath")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/jvm/Classpath.java",
				"--name", name,
				"-t", "mount.resources=configmap:my-deps/sample-1.0.jar@/etc/camel/resources",
				"-t", "jvm.classpath=/etc/camel/resources/my-deps/sample-1.0.jar").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello World!"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, name)()
			jvmTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "jvm")
			g.Expect(jvmTrait).ToNot(BeNil())
			g.Expect(len(jvmTrait)).To(Equal(1))
			g.Expect(jvmTrait["classpath"]).To(Equal("/etc/camel/resources/my-deps/sample-1.0.jar"))
			mountTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "mount")
			g.Expect(mountTrait).ToNot(BeNil())
			g.Expect(len(mountTrait)).To(Equal(1))
			g.Expect(mountTrait["resources"]).To(ContainElements("configmap:my-deps/sample-1.0.jar@/etc/camel/resources"))
		})

		t.Run("JVM trait classpath on specific classpath", func(t *testing.T) {
			name := RandomizedSuffixName("classpath")
			g.Expect(KamelRun(t, ctx, ns,
				"./files/jvm/Classpath.java",
				"--name", name,
				"-t", "mount.resources=configmap:my-deps/sample-1.0.jar@/etc/other/resources",
				"-t", "jvm.classpath=/etc/other/resources/my-deps/sample-1.0.jar").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Hello World!"))

			// check integration schema does not contains unwanted default trait value.
			g.Eventually(UnstructuredIntegration(t, ctx, ns, name)).ShouldNot(BeNil())
			unstructuredIntegration := UnstructuredIntegration(t, ctx, ns, name)()
			jvmTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "jvm")
			g.Expect(jvmTrait).ToNot(BeNil())
			g.Expect(len(jvmTrait)).To(Equal(1))
			g.Expect(jvmTrait["classpath"]).To(Equal("/etc/other/resources/my-deps/sample-1.0.jar"))
			mountTrait, _, _ := unstructured.NestedMap(unstructuredIntegration.Object, "spec", "traits", "mount")
			g.Expect(mountTrait).ToNot(BeNil())
			g.Expect(len(mountTrait)).To(Equal(1))
			g.Expect(mountTrait["resources"]).To(ContainElements("configmap:my-deps/sample-1.0.jar@/etc/other/resources"))
		})
	})
}
