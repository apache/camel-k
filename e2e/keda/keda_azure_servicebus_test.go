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

	. "github.com/apache/camel-k/v2/e2e/support"
)

// There is no Azure Service Bus that the test cluster can reach, and KEDA's azure-servicebus scaler
// queries the Service Bus management API, so these tests check the auto-discovered ScaledObject only.

func TestAzureServiceBusQueueKedaAutoDiscovery(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery Azure Service Bus queue", func(t *testing.T) {
			integrationName := "keda-it-azure-servicebus-queue-auto-discovery"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-azure-servicebus-queue-auto-discovery.yaml", "-n", ns))

			// Wait for ScaledObject
			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			// Verify auto-discovered trigger from the azure-servicebus URI mapping
			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("azure-servicebus"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["queueName"]).To(Equal("my-queue"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["namespace"]).To(Equal("my-namespace"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("topicName"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("connection"))

			ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		})
	})
}

func TestAzureServiceBusTopicKedaAutoDiscovery(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Auto-discovery Azure Service Bus topic", func(t *testing.T) {
			integrationName := "keda-it-azure-servicebus-topic-auto-discovery"

			ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/keda-it-azure-servicebus-topic-auto-discovery.yaml", "-n", ns))

			// Wait for ScaledObject
			g.Eventually(ScaledObject(t, ctx, ns, integrationName), TestTimeoutMedium).
				ShouldNot(BeNil())

			// Verify auto-discovered trigger, with the namespace taken from the component property
			scaledObj := ScaledObject(t, ctx, ns, integrationName)()
			g.Expect(scaledObj).NotTo(BeNil())
			g.Expect(scaledObj.Spec.Triggers).To(HaveLen(1))
			g.Expect(scaledObj.Spec.Triggers[0].Type).To(Equal("azure-servicebus"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["topicName"]).To(Equal("my-topic"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["subscriptionName"]).To(Equal("my-subscription"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata["namespace"]).To(Equal("my-namespace"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("queueName"))
			g.Expect(scaledObj.Spec.Triggers[0].Metadata).NotTo(HaveKey("connection"))

			ExpectExecSucceed(t, g, Kubectl("delete", "it", integrationName, "-n", ns))
		})
	})
}
