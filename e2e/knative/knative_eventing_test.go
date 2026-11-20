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

package knative

import (
	"context"
	"encoding/json"
	"strings"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	corev1 "k8s.io/api/core/v1"
)

type knativeChannelResource struct {
	Name             string `json:"name"`
	Type             string `json:"type"`
	EndpointKind     string `json:"endpointKind"`
	URL              string `json:"url,omitempty"`
	Path             string `json:"path,omitempty"`
	ObjectAPIVersion string `json:"objectApiVersion"`
	ObjectKind       string `json:"objectKind"`
	ObjectName       string `json:"objectName"`
	Reply            bool   `json:"reply"`
}

func knativeChannelConfiguration(channelName, endpointKind, channelURL string) (string, error) {
	resource := knativeChannelResource{
		Name:             channelName,
		Type:             "channel",
		EndpointKind:     endpointKind,
		ObjectAPIVersion: "messaging.knative.dev/v1",
		ObjectKind:       "Channel",
		ObjectName:       channelName,
		Reply:            false,
	}
	if endpointKind == "source" {
		resource.Path = "/channels/" + channelName
	} else {
		resource.URL = channelURL
	}

	configuration, err := json.Marshal(struct {
		Resources []knativeChannelResource `json:"resources"`
	}{
		Resources: []knativeChannelResource{resource},
	})
	if err != nil {
		return "", err
	}

	return string(configuration), nil
}

func TestKnativeEventingChannel(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		const (
			channelName  = "camel-k-eventing"
			consumerName = "knative-channel-consumer"
			producerName = "knative-channel-producer"
			message      = "camel-k-knative-eventing-e2e"
		)

		// The Knative workflow installs the in-memory channel implementation.
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/knative-channel.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl(
			"wait", "--for=condition=Ready", "channel/"+channelName, "--timeout=120s", "-n", ns,
		))

		// The Camel Knative component needs its own environment configuration.
		// This is deliberately supplied directly to the component, not by the deprecated trait.
		channelURL, err := Kubectl(
			"get", "channels.messaging.knative.dev", channelName, "-n", ns, "-o", "jsonpath={.status.address.url}",
		).Output()
		g.Expect(err).NotTo(HaveOccurred())
		g.Expect(strings.TrimSpace(string(channelURL))).NotTo(BeEmpty())

		consumerConfiguration, err := knativeChannelConfiguration(channelName, "source", "")
		g.Expect(err).NotTo(HaveOccurred())

		// Start the consumer first so it is ready before the producer sends its event.
		// Disable the deprecated Knative traits explicitly; the route uses Camel's component.
		g.Expect(KamelRun(t, ctx, ns, "files/knative-channel-consumer.yaml",
			"--name", consumerName,
			"-t", "knative.enabled=false",
			"-t", "knative-service.enabled=false",
			"--env", "CAMEL_KNATIVE_CONFIGURATION="+consumerConfiguration,
		).Execute()).To(Succeed())
		g.Eventually(
			IntegrationConditionStatus(t, ctx, ns, consumerName, v1.IntegrationConditionReady),
			TestTimeoutMedium,
		).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, consumerName)).
			Should(Equal(corev1.PodRunning))

		// With the deprecated trait disabled, create the Subscription explicitly.
		// It targets the Kubernetes Service created for the consumer Integration.
		ExpectExecSucceed(t, g, Kubectl(
			"apply", "-f", "files/knative-channel-subscription.yaml", "-n", ns,
		))
		ExpectExecSucceed(t, g, Kubectl(
			"wait", "--for=condition=Ready", "subscription/"+channelName+"-"+consumerName, "--timeout=120s", "-n", ns,
		))

		producerConfiguration, err := knativeChannelConfiguration(
			channelName, "sink", strings.TrimSpace(string(channelURL)),
		)
		g.Expect(err).NotTo(HaveOccurred())

		g.Expect(KamelRun(t, ctx, ns, "files/knative-channel-producer.yaml",
			"--name", producerName,
			"-t", "knative.enabled=false",
			"-t", "knative-service.enabled=false",
			"--env", "CAMEL_KNATIVE_CONFIGURATION="+producerConfiguration,
		).Execute()).To(Succeed())
		g.Eventually(
			IntegrationConditionStatus(t, ctx, ns, producerName, v1.IntegrationConditionReady), TestTimeoutMedium,
		).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationPodPhase(t, ctx, ns, producerName)).
			Should(Equal(corev1.PodRunning))

		// Verify that the event sent to the channel reaches the consumer Integration.
		g.Eventually(IntegrationLogs(t, ctx, ns, consumerName)).
			Should(ContainSubstring(message))
	})
}
