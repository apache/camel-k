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

	. "github.com/apache/camel-k/v2/e2e/support"
)

func TestArkMQKedaAutoDiscovery(t *testing.T) {
	WithExistingNamedTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery ArkMQ queue", func(t *testing.T) {
			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-arkmq-to-log-auto.yaml"))
			integrationName := "keda-arkmq-to-log-auto"

			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("artemis-queue"))
			// Auto-discovered from the amqp:queue URI
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["queueName"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerAddress"]).To(Equal("my-queue"))
			// Merged from autoMetadata
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["managementEndpoint"]).To(Equal("my-broker-hdls-svc.arkmq.svc:8161"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["brokerName"]).To(Equal("amq-broker"))

			ExpectExecSucceed(t, g, Kubectl("delete", "pipe", integrationName, "-n", ns))
		})
	}, "arkmq")
}
