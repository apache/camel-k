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
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestRunBuildMaxParallelPipelines(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		InstallOperatorWithConf(t, ctx, g, ns, "", false, map[string]string{
			"MAX_RUNNING_BUILDS": "1",
		})
		integrationA := RandomizedSuffixName("java-a")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
			"--name", integrationA,
		).Execute()).To(Succeed())

		// The presence of a builder property guarantee a new build
		integrationB := RandomizedSuffixName("java-b")
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java",
			"--name", integrationB,
			"-t", "builder.properties=build-property=new",
		).Execute()).To(Succeed())

		g.Eventually(Builds(t, ctx, ns)).Should(Equal(2))
		// At least one build starts running
		g.Eventually(BuildsRunning(t, ctx, ns)).Should(Equal(1))
		// Never more than one build running concurrently
		g.Consistently(BuildsRunning(t, ctx, ns), "30s", "2s").
			Should(BeNumerically("<=", 1))
		// Eventually all builds complete
		g.Eventually(BuildsRunning(t, ctx, ns), TestTimeoutLong).
			Should(Equal(0))

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationA, v1.IntegrationConditionReady)).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationB, v1.IntegrationConditionReady)).
			Should(Equal(corev1.ConditionTrue))
	})
}
