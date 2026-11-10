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

package keda

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestPulsarKedaAutoscale(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/pulsar-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment", "pulsar", "-n", ns, "--timeout=300s"))

		t.Run("Pulsar auto-discovered trigger", func(t *testing.T) {
			consumerName := "keda-it-pulsar-to-log-auto"
			producerName := "timer-to-pulsar"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", cloneAndReplaceNamespace(t, "files/keda-it-pulsar-to-log-auto.yaml", ns), "-n", ns))

			// The ScaledObject uses the trigger discovered from the Pulsar endpoint URI
			g.Eventually(ScaledObject(t, ctx, ns, consumerName), TestTimeoutMedium).ShouldNot(BeNil())
			scaledObj := ScaledObject(t, ctx, ns, consumerName)()
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("pulsar"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["topic"]).To(Equal("persistent://public/default/my-topic"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["subscription"]).To(Equal("my-subscription"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["adminURL"]).To(Equal("http://pulsar." + ns + ".svc:8080"))

			// Start consumer; it creates the subscription
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Scale down to 0 when idle
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))

			// Start producer to build a backlog
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", cloneAndReplaceNamespace(t, "files/timer-to-pulsar.yaml", ns), "-n", ns))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutLong).
				Should(Equal(corev1.ConditionTrue))

			// Consumer scales up to 1 and processes the messages
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 1)))
			g.Eventually(IntegrationLogs(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(ContainSubstring("Hello Pulsar"))

			// Stop producer and scale back to 0
			ExpectExecSucceed(t, g, Kubectl("delete", "it", producerName, "-n", ns))
			g.Eventually(IntegrationStatusReplicas(t, ctx, ns, consumerName), TestTimeoutMedium).
				Should(PointTo(BeNumerically("==", 0)))
		})
	})
}

// cloneAndReplaceNamespace clones and replace the content marked as %%% with the namespace passed as parameter.
func cloneAndReplaceNamespace(t *testing.T, srcPath, namespace string) string {
	t.Helper()

	tempPath := filepath.Join(t.TempDir(), filepath.Base(srcPath))

	content, err := os.ReadFile(srcPath)
	if err != nil {
		t.Fatalf("failed to read src file: %v", err)
	}
	updated := strings.ReplaceAll(string(content), "%%%", namespace)
	if err := os.WriteFile(tempPath, []byte(updated), 0o644); err != nil {
		t.Fatalf("failed to write dst file: %v", err)
	}

	return tempPath
}
