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

package keda

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestIBMMQKedaAutoDiscovery(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		integrationName := "keda-it-ibmmq-to-log-auto"

		// Always remove the Integration and disposable broker, including when
		// an assertion fails part-way through the test.
		t.Cleanup(func() {
			_ = Kubectl("delete", "it", integrationName, "-n", ns, "--ignore-not-found=true").Run()
			_ = Kubectl("delete", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns, "--ignore-not-found=true").Run()
		})

		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("rollout", "status", "deployment/ibmmq", "-n", ns, "--timeout=300s"))

		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-ibmmq-to-log-auto.yaml", "-n", ns))

		// Wait until Camel K has reconciled the Integration and created the
		// ScaledObject from the jms:queue endpoint.
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, integrationName, v1.IntegrationConditionReady), TestTimeoutLong).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutLong).
			ShouldNot(BeNil())

		// Ready confirms that KEDA accepted the scaler configuration.
		ExpectExecSucceed(t, g, Kubectl("wait", "--for=condition=Ready", "scaledobject/"+integrationName, "-n", ns, "--timeout=180s"))

		// ScalingActive confirms that KEDA can query the live IBM MQ REST
		// endpoint with the Secret-backed username and password.
		ExpectExecSucceed(t, g, Kubectl("wait", "--for=condition=ScalingActive", "hpa/keda-hpa-"+integrationName, "-n", ns, "--timeout=180s"))

		scaledObject := ScaledObject(t, ctx, ns, integrationName)()
		g.Expect(scaledObject).NotTo(BeNil())
		g.Expect(scaledObject.Spec.Triggers).To(HaveLen(1))
		g.Expect(scaledObject.Spec.Triggers[0].Type).To(Equal("ibmmq"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("queueName", "DEV.QUEUE.1"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("host", "https://ibmmq.keda.svc:9443/ibmmq/rest/v2/admin/action/qmgr/QM1/mqsc"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("queueDepth", "1"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).To(HaveKeyWithValue("unsafeSsl", "true"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).NotTo(HaveKey("username"))
		g.Expect(scaledObject.Spec.Triggers[0].Metadata).NotTo(HaveKey("password"))

		ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("delete", "-f", "files/ibmmq-ephemeral.yaml", "-n", ns))
	}, "keda")
}
