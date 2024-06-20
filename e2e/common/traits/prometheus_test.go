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
	"fmt"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	monitoringv1 "github.com/prometheus-operator/prometheus-operator/pkg/apis/monitoring/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

func TestPrometheusTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ocp, err := openshift.IsOpenShift(TestClient(t))
		require.NoError(t, err)
		// Do not create PodMonitor for the time being as CI test runs on OCP 3.11
		createPodMonitor := false
		g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "-t", "prometheus.enabled=true", "-t", fmt.Sprintf("prometheus.pod-monitor=%v", createPodMonitor)).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		t.Run("Metrics endpoint works", func(t *testing.T) {
			pod := IntegrationPod(t, ctx, ns, "java")
			response, err := TestClient(t).CoreV1().RESTClient().Get().
				Timeout(30 * time.Second).
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/q/metrics", ns, pod().Name)).DoRaw(ctx)
			if err != nil {
				assert.Fail(t, err.Error())
			}
			assert.Contains(t, string(response), "camel_exchanges_total")
		})

		if ocp && createPodMonitor {
			t.Run("PodMonitor is created", func(t *testing.T) {
				sm := podMonitor(t, ctx, ns, "java")
				g.Eventually(sm, TestTimeoutShort).ShouldNot(BeNil())
			})
		}
	})
}

func podMonitor(t *testing.T, ctx context.Context, ns string, name string) func() *monitoringv1.PodMonitor {
	return func() *monitoringv1.PodMonitor {
		pm := monitoringv1.PodMonitor{}
		key := ctrl.ObjectKey{
			Namespace: ns,
			Name:      name,
		}
		err := TestClient(t).Get(ctx, key, &pm)
		if err != nil && k8serrors.IsNotFound(err) {
			return nil
		} else if err != nil {
			panic(err)
		}
		return &pm
	}
}
