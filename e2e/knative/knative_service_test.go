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

	. "github.com/apache/camel-k/v2/e2e/support"
	camelv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	v1 "k8s.io/api/core/v1"
)

func TestKnativeServiceURL(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {

		t.Run("Service endpoint url check", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/knativeurl1.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativeurl1"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "knativeurl1", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
			ks := KnativeService(t, ctx, ns, "knativeurl1")
			g.Eventually(ks, TestTimeoutShort).ShouldNot(BeNil())
			url := "http://knativeurl1." + ns + ".svc.cluster.local"
			g.Eventually(ks().Status.RouteStatusFields.URL.String(), TestTimeoutShort).Should(Equal(url))
		})

		t.Run("Service multiple endpoint url check", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/knativeurl2.yaml").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "knativeurl2"), TestTimeoutLong).Should(Equal(v1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, "knativeurl2", camelv1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(v1.ConditionTrue))
			ks := KnativeService(t, ctx, ns, "knativeurl2")
			g.Eventually(ks, TestTimeoutShort).ShouldNot(BeNil())
			url := "http://knativeurl2." + ns + ".svc.cluster.local"
			g.Eventually(ks().Status.RouteStatusFields.URL.String(), TestTimeoutShort).Should(Equal(url))
		})
		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}
