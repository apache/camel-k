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

package networkpolicy

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestServiceNetworkPolicy(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		namespace, err := TestClient(t).CoreV1().Namespaces().Get(ctx, ns, metav1.GetOptions{})
		g.Expect(err).ToNot(HaveOccurred())

		labels := namespace.Labels
		if labels == nil {
			labels = map[string]string{}
		}
		labels["network-policy-test"] = "enabled"
		namespace.Labels = labels
		_, err = TestClient(t).CoreV1().Namespaces().Update(ctx, namespace, metav1.UpdateOptions{})
		g.Expect(err).ToNot(HaveOccurred())

		serverName := RandomizedSuffixName("network-policy-server")
		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
			"-t", "service.enabled=true",
			"-t", "service.network-policy-enabled=true",
			"-t", "service.network-policy-namespace-selector.'network-policy-test'=enabled",
			"-t", "service.network-policy-pod-selector.app=allowed",
			"--name", serverName,
		).Execute()).To(Succeed())

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, serverName, v1.IntegrationConditionReady), TestTimeoutMedium).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(NetworkPolicyByName(t, ctx, ns, serverName), TestTimeoutShort).ShouldNot(BeNil())

		allowedName := RandomizedSuffixName("network-policy-allowed")
		deniedName := RandomizedSuffixName("network-policy-denied")

		g.Expect(KamelRun(t, ctx, ns, "files/NetworkPolicyConsumer.java",
			"-p", "serviceName="+serverName,
			"--label", "app=allowed",
			"-t", "owner.target-labels=*",
			"--name", allowedName,
		).Execute()).To(Succeed())

		g.Expect(KamelRun(t, ctx, ns, "files/NetworkPolicyConsumer.java",
			"-p", "serviceName="+serverName,
			"--label", "app=denied",
			"-t", "owner.target-labels=*",
			"--name", deniedName,
		).Execute()).To(Succeed())

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, allowedName, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, deniedName, v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))

		allowedPod := IntegrationPod(t, ctx, ns, allowedName)()
		deniedPod := IntegrationPod(t, ctx, ns, deniedName)()
		g.Expect(allowedPod.GetLabels()["app"]).To(Equal("allowed"))
		g.Expect(deniedPod.GetLabels()["app"]).To(Equal("denied"))

		g.Eventually(IntegrationLogs(t, ctx, ns, allowedName), TestTimeoutMedium).
			Should(ContainSubstring("ACCESS_GRANTED"))
		g.Eventually(IntegrationLogs(t, ctx, ns, deniedName), TestTimeoutMedium).
			Should(ContainSubstring("ACCESS_DENIED"))
	})
}

func NetworkPolicyByName(t *testing.T, ctx context.Context, namespace, name string) func() *networkingv1.NetworkPolicy {
	return func() *networkingv1.NetworkPolicy {
		policy := &networkingv1.NetworkPolicy{}
		err := TestClient(t).Get(ctx, ctrl.ObjectKey{Name: name, Namespace: namespace}, policy)
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				t.Logf("failed to read NetworkPolicy %s/%s: %v", namespace, name, err)
			}
			return nil
		}
		return policy
	}
}
