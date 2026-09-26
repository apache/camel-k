//go:build integration
// +build integration

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

package arkmq

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestArkMQ(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, arkmqNs string) {
		t.Run("ArkMQ ActiveMQArtemisAddress resource", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/timer-to-arkmq.yaml"))
			// Wait for the readiness of the Integration
			g.Eventually(IntegrationConditionStatus(t, ctx, arkmqNs, "timer-to-arkmq", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/arkmq-to-log.yaml"))
			g.Eventually(IntegrationConditionStatus(t, ctx, arkmqNs, "arkmq-to-log", v1.IntegrationConditionReady), TestTimeoutMedium).
				Should(Equal(corev1.ConditionTrue))
			// Verify we are consuming records from the queue
			g.Eventually(IntegrationLogs(t, ctx, arkmqNs, "arkmq-to-log")).Should(ContainSubstring("Body is null"))

			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", arkmqNs).Execute()).To(Succeed())
		})
	}, "arkmq")
}
