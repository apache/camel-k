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

package advanced

import (
	"context"
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestEnvironmentTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// HTTP proxy configuration
		httpProxy := "http://proxy"
		noProxy := []string{
			".cluster.local",
			".svc",
			"localhost",
			".apache.org",
		}

		// Retrieve the Kubernetes Service ClusterIPs to populate the NO_PROXY environment variable
		svc := Service(t, ctx, TestDefaultNamespace, "kubernetes")()
		g.Expect(svc).NotTo(BeNil())

		noProxy = append(noProxy, svc.Spec.ClusterIPs...)

		// Retrieve the internal container registry to populate the NO_PROXY environment variable
		if registry, ok := os.LookupEnv("KAMEL_INSTALL_REGISTRY"); ok {
			domain := RegistryRegexp.FindString(registry)
			g.Expect(domain).NotTo(BeNil())
			domain = strings.Split(domain, ":")[0]
			noProxy = append(noProxy, domain)
		}

		// Install Camel K with the HTTP proxy environment variable
		//g.Expect(InstallOperator(t, ctx, operatorID, ns, "--operator-env-vars", fmt.Sprintf("HTTP_PROXY=%s", httpProxy), "--operator-env-vars", "NO_PROXY="+strings.Join(noProxy, ","))).To(Succeed())
		InstallOperator(t, g, ns)

		t.Run("Run integration with default environment", func(t *testing.T) {
			name := RandomizedSuffixName("java-default")
			g.Expect(KamelRun(t, ctx, ns, "--name", name, "files/Java.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Expect(IntegrationPod(t, ctx, ns, name)()).To(WithTransform(podEnvVars, And(
				ContainElement(corev1.EnvVar{Name: "CAMEL_K_VERSION", Value: defaults.Version}),
				ContainElement(corev1.EnvVar{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				}}),
				ContainElement(corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				}}),
				ContainElement(corev1.EnvVar{Name: "HTTP_PROXY", Value: httpProxy}),
				ContainElement(corev1.EnvVar{Name: "NO_PROXY", Value: strings.Join(noProxy, ",")}),
			)))
		})

		t.Run("Run integration with custom environment", func(t *testing.T) {
			name := RandomizedSuffixName("java-custom-proxy")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "environment.vars=HTTP_PROXY=http://custom.proxy").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Expect(IntegrationPod(t, ctx, ns, name)()).To(WithTransform(podEnvVars, And(
				ContainElement(corev1.EnvVar{Name: "CAMEL_K_VERSION", Value: defaults.Version}),
				ContainElement(corev1.EnvVar{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				}}),
				ContainElement(corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				}}),
				ContainElement(corev1.EnvVar{Name: "HTTP_PROXY", Value: "http://custom.proxy"}),
				ContainElement(corev1.EnvVar{Name: "NO_PROXY", Value: strings.Join(noProxy, ",")}),
			)))
		})

		t.Run("Run integration without default HTTP proxy environment", func(t *testing.T) {
			name := RandomizedSuffixName("java-no-proxy")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name, "-t", "environment.http-proxy=false").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Expect(IntegrationPod(t, ctx, ns, name)()).To(WithTransform(podEnvVars, And(
				ContainElement(corev1.EnvVar{Name: "CAMEL_K_VERSION", Value: defaults.Version}),
				ContainElement(corev1.EnvVar{Name: "NAMESPACE", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.namespace",
					},
				}}),
				ContainElement(corev1.EnvVar{Name: "POD_NAME", ValueFrom: &corev1.EnvVarSource{
					FieldRef: &corev1.ObjectFieldSelector{
						APIVersion: "v1",
						FieldPath:  "metadata.name",
					},
				}}),
				Not(ContainElement(corev1.EnvVar{Name: "HTTP_PROXY", Value: httpProxy})),
				Not(ContainElement(corev1.EnvVar{Name: "NO_PROXY", Value: strings.Join(noProxy, ",")})),
			)))
		})
	})
}

func podEnvVars(pod *corev1.Pod) []corev1.EnvVar {
	for _, container := range pod.Spec.Containers {
		if container.Name == "integration" {
			return container.Env
		}
	}
	return nil
}
