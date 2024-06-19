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
	"regexp"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
)

func TestKamelCliDescribe(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		g.Expect(KamelRun(t, ctx, ns, "files/yaml.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

		t.Run("Test kamel describe integration", func(t *testing.T) {
			integration := GetOutputString(Kamel(t, ctx, "describe", "integration", "yaml", "-n", ns))
			r, _ := regexp.Compile("(?sm).*Name:\\s+yaml.*")
			g.Expect(integration).To(MatchRegexp(r.String()))

			r, _ = regexp.Compile("(?sm).*Phase:\\s+Running.*")
			g.Expect(integration).To(MatchRegexp(r.String()))

			g.Expect(integration).To(ContainSubstring("Dependencies:"))
			g.Expect(integration).To(ContainSubstring("Conditions:"))
		})

		t.Run("Test kamel describe integration kit", func(t *testing.T) {
			kitName := Integration(t, ctx, ns, "yaml")().Status.IntegrationKit.Name
			kitNamespace := Integration(t, ctx, ns, "yaml")().Status.IntegrationKit.Namespace
			kit := GetOutputString(Kamel(t, ctx, "describe", "kit", kitName, "-n", kitNamespace))

			r, _ := regexp.Compile("(?sm).*Namespace:\\s+" + kitNamespace + ".*")
			g.Expect(kit).To(MatchRegexp(r.String()))

			r, _ = regexp.Compile("(?sm).*Runtime Version:\\s+" + defaults.DefaultRuntimeVersion + ".*")
			g.Expect(kit).To(MatchRegexp(r.String()))

			g.Expect(kit).To(ContainSubstring("camel-quarkus-core"))

			g.Expect(kit).To(ContainSubstring("Artifacts:"))
			g.Expect(kit).To(ContainSubstring("Dependencies:"))
		})

	})
}
