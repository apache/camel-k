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

package misc

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gstruct"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestPipeWithImage(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		operatorID := "camel-k-pipe-image"
		g.Expect(CopyCamelCatalog(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ctx, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, ctx, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ctx, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		bindingID := "with-image-binding"

		t.Run("run with initial image", func(t *testing.T) {
			expectedImage := "docker.io/jmalloc/echo-server:0.3.2"

			g.Expect(KamelBindWithID(t, ctx, operatorID, ns, "my-own-timer-source", "my-own-log-sink", "--annotation", "trait.camel.apache.org/container.image="+expectedImage, "--annotation", "trait.camel.apache.org/jvm.enabled=false", "--annotation", "trait.camel.apache.org/kamelets.enabled=false", "--annotation", "trait.camel.apache.org/dependencies.enabled=false", "--annotation", "test=1", "--name", bindingID).Execute()).To(Succeed())

			g.Eventually(IntegrationGeneration(t, ctx, ns, bindingID)).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(Integration(t, ctx, ns, bindingID)).Should(WithTransform(Annotations, And(
				HaveKeyWithValue("test", "1"),
				HaveKeyWithValue("trait.camel.apache.org/container.image", expectedImage),
			)))
			g.Eventually(IntegrationStatusImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, bindingID), TestTimeoutLong).
				Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
		})

		t.Run("run with new image", func(t *testing.T) {
			expectedImage := "docker.io/jmalloc/echo-server:0.3.3"

			g.Expect(KamelBindWithID(t, ctx, operatorID, ns, "my-own-timer-source", "my-own-log-sink", "--annotation", "trait.camel.apache.org/container.image="+expectedImage, "--annotation", "trait.camel.apache.org/jvm.enabled=false", "--annotation", "trait.camel.apache.org/kamelets.enabled=false", "--annotation", "trait.camel.apache.org/dependencies.enabled=false", "--annotation", "test=2", "--name", bindingID).Execute()).To(Succeed())
			g.Eventually(IntegrationGeneration(t, ctx, ns, bindingID)).
				Should(gstruct.PointTo(BeNumerically("==", 1)))
			g.Eventually(Integration(t, ctx, ns, bindingID)).Should(WithTransform(Annotations, And(
				HaveKeyWithValue("test", "2"),
				HaveKeyWithValue("trait.camel.apache.org/container.image", expectedImage),
			)))
			g.Eventually(IntegrationStatusImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, bindingID), TestTimeoutLong).
				Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationPodImage(t, ctx, ns, bindingID)).
				Should(Equal(expectedImage))
		})

		g.Expect(CamelK(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})

}
