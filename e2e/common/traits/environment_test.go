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
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestEnvironmentTrait(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("custom variables and container metadata are available to the route", func(t *testing.T) {
			name := RandomizedSuffixName("environment-enabled")
			customValue := "environment-trait-e2e-value"

			g.Expect(KamelRun(t, ctx, ns, "files/EnvironmentTrait.java",
				"--name", name,
				"-t", "environment.container-meta=true",
				"-t", "environment.vars=CAMEL_K_TEST_VALUE="+customValue,
			).Execute()).To(Succeed())

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod).NotTo(BeNil())
			g.Expect(pod.Spec.Containers).To(HaveLen(1))

			envVars := make(map[string]corev1.EnvVar)
			for _, envVar := range pod.Spec.Containers[0].Env {
				envVars[envVar.Name] = envVar
			}

			g.Expect(envVars).To(HaveKey("NAMESPACE"))
			g.Expect(envVars["NAMESPACE"].ValueFrom).NotTo(BeNil())
			g.Expect(envVars["NAMESPACE"].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.namespace"))
			g.Expect(envVars).To(HaveKey("POD_NAME"))
			g.Expect(envVars["POD_NAME"].ValueFrom).NotTo(BeNil())
			g.Expect(envVars["POD_NAME"].ValueFrom.FieldRef.FieldPath).To(Equal("metadata.name"))
			g.Expect(envVars).To(HaveKeyWithValue("CAMEL_K_TEST_VALUE", HaveField("Value", customValue)))

			expectedMessage := "namespace=" + ns + ";pod=" + pod.Name + ";integration=" + name + ";custom=" + customValue
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutMedium).Should(ContainSubstring(expectedMessage))
		})

		t.Run("container metadata and proxy propagation can be disabled", func(t *testing.T) {
			name := RandomizedSuffixName("environment-disabled")
			customValue := "environment-trait-disabled-value"

			g.Expect(KamelRun(t, ctx, ns, "files/EnvironmentTrait.java",
				"--name", name,
				"-t", "environment.container-meta=false",
				"-t", "environment.http-proxy=false",
				"-t", "environment.vars=CAMEL_K_TEST_VALUE="+customValue,
			).Execute()).To(Succeed())

			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			pod := IntegrationPod(t, ctx, ns, name)()
			g.Expect(pod).NotTo(BeNil())
			g.Expect(pod.Spec.Containers).To(HaveLen(1))

			envNames := make(map[string]struct{})
			for _, envVar := range pod.Spec.Containers[0].Env {
				envNames[envVar.Name] = struct{}{}
			}

			g.Expect(envNames).NotTo(HaveKey("NAMESPACE"))
			g.Expect(envNames).NotTo(HaveKey("POD_NAME"))
			g.Expect(envNames).NotTo(HaveKey("HTTP_PROXY"))
			g.Expect(envNames).NotTo(HaveKey("HTTPS_PROXY"))
			g.Expect(envNames).NotTo(HaveKey("NO_PROXY"))
			g.Expect(pod.Spec.Containers[0].Env).To(ContainElement(corev1.EnvVar{
				Name:  "CAMEL_K_TEST_VALUE",
				Value: customValue,
			}))

			expectedMessage := "custom=" + customValue
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutMedium).Should(ContainSubstring(expectedMessage))
		})
	})
}
