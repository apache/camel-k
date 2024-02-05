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
	"context"
	"testing"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestLoggingCategory(t *testing.T) {
	t.Parallel()

	// WithNewTestNamespace(t, func(g *WithT, ns string) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {

		operatorID := "camel-k-traits-logging"

		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		t.Run("Test debug log level", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name, "-t", "logging.category.'org.apache.camel.impl'=DEBUG").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Adding routes from builder"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})

		t.Run("Test multiple log levels", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRunWithID(t, ctx, operatorID, ns, "files/Java.java", "--name", name,
				"-t", "logging.category.'org.apache.camel.impl'=DEBUG",
				"-t", "logging.category.'io.quarkus'=DEBUG").Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Adding routes from builder"))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Vert.x Cache configured to"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).Should(BeNil())
		})
	})
}
