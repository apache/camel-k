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

package cli

import (
	"context"
	"fmt"
	"testing"

	corev1 "k8s.io/api/core/v1"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKamelCLIGet(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {

		t.Run("get integration", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			// regex is used for the compatibility of tests between OC and vanilla K8
			// kamel get may have different output depending on the platform
			g.Eventually(IntegrationKit(t, ctx, ns, "yaml")).ShouldNot(Equal(""))
			kitName := IntegrationKit(t, ctx, ns, "yaml")()
			kitNamespace := IntegrationKitNamespace(t, ctx, ns, "yaml")()
			regex := fmt.Sprintf("^NAME\tPHASE\tKIT\n\\s*yaml\tRunning\t(%s/%s|%s)", kitNamespace, kitName, kitName)
			g.Eventually(GetOutputString(Kamel(t, ctx, "get", "-n", ns))).Should(MatchRegexp(regex))
		})

		t.Run("get several integrations", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

			g.Eventually(IntegrationKit(t, ctx, ns, "java")).ShouldNot(Equal(""))
			g.Eventually(IntegrationKit(t, ctx, ns, "yaml")).ShouldNot(Equal(""))
			kitName1 := IntegrationKit(t, ctx, ns, "java")()
			kitName2 := IntegrationKit(t, ctx, ns, "yaml")()
			kitNamespace1 := IntegrationKitNamespace(t, ctx, ns, "java")()
			kitNamespace2 := IntegrationKitNamespace(t, ctx, ns, "yaml")()
			regex := fmt.Sprintf("^NAME\tPHASE\tKIT\n\\s*java\tRunning\t"+
				"(%s/%s|%s)\n\\s*yaml\tRunning\t(%s/%s|%s)\n", kitNamespace1, kitName1, kitName1, kitNamespace2, kitName2, kitName2)
			g.Eventually(GetOutputString(Kamel(t, ctx, "get", "-n", ns))).Should(MatchRegexp(regex))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("get no integrations", func(t *testing.T) {
			g.Expect(GetOutputString(Kamel(t, ctx, "get", "-n", ns))).NotTo(ContainSubstring("Running"))
			g.Expect(GetOutputString(Kamel(t, ctx, "get", "-n", ns))).NotTo(ContainSubstring("Building Kit"))
		})
	})
}
