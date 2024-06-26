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

package knative

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/knative"
)

func TestKnativeServiceURL(t *testing.T) {
	g := NewWithT(t)

	installed, err := knative.IsServingInstalled(TestClient(t))
	g.Expect(err).NotTo(HaveOccurred())
	if !installed {
		t.Error("Knative not installed in the cluster")
		t.FailNow()
	}
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k"
		// Install without profile (should automatically detect the presence of Knative)
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())
		g.Eventually(PlatformPhase(t, ctx, ns), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))
		g.Eventually(PlatformProfile(t, ctx, ns), TestTimeoutShort).Should(Equal(v1.TraitProfile("")))
		cluster := Platform(t, ctx, ns)().Status.Cluster

		t.Run("run yaml on cluster profile", func(t *testing.T) {
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/knative2.yaml", "--profile", string(cluster)).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "knative2"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "knative2"), TestTimeoutShort).Should(ContainSubstring("http://knative2.ns.domain"))
			g.Eventually(IntegrationTraitProfile(t, ctx, ns, "knative2"), TestTimeoutShort).Should(Equal(v1.TraitProfile(string(cluster))))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})
	})
}
