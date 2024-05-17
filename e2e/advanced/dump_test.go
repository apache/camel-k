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
	"fmt"
	"testing"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestKamelCLIDump(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("dump empty namespace", func(t *testing.T) {
			dump := GetOutputString(Kamel(t, ctx, "dump", "-n", ns))

			g.Expect(dump).To(ContainSubstring("Found 0 integrations:"))
			g.Expect(dump).To(ContainSubstring("Found 0 deployments:"))
		})

		t.Run("dump non-empty namespace", func(t *testing.T) {
			operatorID := fmt.Sprintf("camel-k-%s", ns)
			g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
			g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
			g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
			g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "yaml")).Should(ContainSubstring("Magicstring!"))

			dump := GetOutputString(Kamel(t, ctx, "dump", "-n", ns))
			g.Expect(dump).To(ContainSubstring("Found 1 platforms"))
			g.Expect(dump).To(ContainSubstring("Found 1 integrations"))
			g.Expect(dump).To(ContainSubstring("name: yaml"))
			g.Expect(dump).To(ContainSubstring("Magicstring!"))
		})
	})
}
