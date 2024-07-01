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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestMasterTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("master works", func(t *testing.T) {
			g.Expect(KamelRun(t, ctx, ns, "files/Master.java").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "master"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "master"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("only one integration with master runs", func(t *testing.T) {
			nameFirst := RandomizedSuffixName("first")
			g.Expect(KamelRun(t, ctx, ns, "files/Master.java", "--name", nameFirst,
				"--label", "leader-group=same", "-t", "master.label-key=leader-group", "-t", "master.label-value=same", "-t", "owner.target-labels=leader-group",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameFirst, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameFirst), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			// Start a second integration with the same lock (it should not start the route before 15 seconds)
			nameSecond := RandomizedSuffixName("second")
			g.Expect(KamelRun(t, ctx, ns, "files/Master.java", "--name", nameSecond,
				"--label", "leader-group=same", "-t", "master.label-key=leader-group", "-t", "master.label-value=same", "-t", "owner.target-labels=leader-group",
				"-t", fmt.Sprintf("master.resource-name=%s-lock", nameFirst),
			).Execute()).To(Succeed())
			g.Eventually(IntegrationLogs(t, ctx, ns, nameSecond), TestTimeoutShort).Should(ContainSubstring("started in"))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameSecond), 15*time.Second).ShouldNot(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameSecond, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		})
	})
}
