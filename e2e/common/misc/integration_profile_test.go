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

func TestIntegrationProfileDependencies(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		integrationProfile := v1.NewIntegrationProfile(ns, "my-profile")
		// Any catalog dependency is good enough for this test as we just want to verify that any dependency set in
		// the profile is also added in the Integration status (which means it was added to maven build project).
		integrationProfile.Spec.Dependencies = []string{"camel:zipfile"}

		g.Expect(CreateIntegrationProfile(t, ctx, &integrationProfile)).To(Succeed())

		t.Run("Run sample integration", func(t *testing.T) {
			name := RandomizedSuffixName("profile")
			g.Expect(KamelRun(t, ctx, ns,
				"--name", name,
				"--annotation", "camel.apache.org/integration-profile.id=my-profile",
				"files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutShort).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Eventually(func() []string {
				return Integration(t, ctx, ns, name)().Status.Dependencies
			}, TestTimeoutMedium).Should(ContainElement("camel:zipfile"))
		})
	})
}
