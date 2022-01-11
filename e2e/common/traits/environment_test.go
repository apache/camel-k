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
	"os"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/e2e/support"
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/util/defaults"
)

func TestEnvironmentTrait(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		// HTTP proxy configuration
		httpProxy := "http://proxy"
		noProxy := []string{
			".cluster.local",
			".svc",
			"localhost",
			".apache.org",
		}

		// Retrieve the Kubernetes Service ClusterIPs to populate the NO_PROXY environment variable
		svc := Service("default", "kubernetes")()
		Expect(svc).NotTo(BeNil())

		noProxy = append(noProxy, svc.Spec.ClusterIPs...)

		// Retrieve the internal container registry to populate the NO_PROXY environment variable
		if registry, ok := os.LookupEnv("KAMEL_INSTALL_REGISTRY"); ok {
			domain := RegistryRegexp.FindString(registry)
			Expect(domain).NotTo(BeNil())
			domain = strings.Split(domain, ":")[0]
			noProxy = append(noProxy, domain)
		}

		// Install Camel K with the HTTP proxy environment variable
		Expect(Kamel("install", "-n", ns,
			"--operator-env-vars", fmt.Sprintf("HTTP_PROXY=%s", httpProxy),
			"--operator-env-vars", "NO_PROXY="+strings.Join(noProxy, ","),
		).Execute()).To(Succeed())

		t.Run("Run integration with default environment", func(t *testing.T) {
			Expect(Kamel("run", "-n", ns, "files/Java.java").Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Expect(IntegrationPod(ns, "java")()).To(WithTransform(podEnvVars, And(
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
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"-t", "environment.vars=\"HTTP_PROXY=http://custom.proxy\"",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Expect(IntegrationPod(ns, "java")()).To(WithTransform(podEnvVars, And(
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
			Expect(Kamel("run", "-n", ns, "files/Java.java",
				"-t", "environment.http-proxy=false",
			).Execute()).To(Succeed())
			Eventually(IntegrationPodPhase(ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			Eventually(IntegrationConditionStatus(ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			Eventually(IntegrationLogs(ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			Expect(IntegrationPod(ns, "java")()).To(WithTransform(podEnvVars, And(
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

		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
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
