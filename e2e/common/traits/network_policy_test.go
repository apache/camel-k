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

package common

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
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

		name := RandomizedSuffixName("network-policy-server")
		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
			"-t", "service.enabled=true",
			"-t", "service.network-policy-enabled=true",
			"-t", "service.network-policy-namespace-selector.'network-policy-test'=enabled",
			"-t", "service.network-policy-pod-selector.app=allowed",
			"--name", name,
		).Execute()).To(Succeed())

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(NetworkPolicyByName(t, ctx, ns, name), TestTimeoutShort).ShouldNot(BeNil())

		allowedPod := networkPolicyClientPod(name+"-allowed", ns, name, "allowed")
		deniedPod := networkPolicyClientPod(name+"-denied", ns, name, "denied")
		_, err = TestClient(t).CoreV1().Pods(ns).Create(ctx, allowedPod, metav1.CreateOptions{})
		g.Expect(err).ToNot(HaveOccurred())
		_, err = TestClient(t).CoreV1().Pods(ns).Create(ctx, deniedPod, metav1.CreateOptions{})
		g.Expect(err).ToNot(HaveOccurred())

		t.Cleanup(func() {
			_ = TestClient(t).CoreV1().Pods(ns).Delete(ctx, allowedPod.Name, metav1.DeleteOptions{})
			_ = TestClient(t).CoreV1().Pods(ns).Delete(ctx, deniedPod.Name, metav1.DeleteOptions{})
		})

		g.Eventually(PodPhase(t, ctx, ns, allowedPod.Name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))
		g.Eventually(PodPhase(t, ctx, ns, deniedPod.Name), TestTimeoutMedium).Should(Equal(corev1.PodRunning))

		g.Eventually(Logs(t, ctx, ns, allowedPod.Name, corev1.PodLogOptions{}), TestTimeoutMedium).
			Should(ContainSubstring("ACCESS_GRANTED"))
		g.Eventually(Logs(t, ctx, ns, deniedPod.Name, corev1.PodLogOptions{}), TestTimeoutMedium).
			Should(ContainSubstring("ACCESS_DENIED"))
	})
}

func networkPolicyClientPod(name, namespace, serviceName, app string) *corev1.Pod {
	return &corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Pod",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: namespace,
			Labels: map[string]string{
				"app": app,
			},
		},
		Spec: corev1.PodSpec{
			RestartPolicy: corev1.RestartPolicyAlways,
			Containers: []corev1.Container{
				{
					Name:            "client",
					Image:           "busybox:1.36.1",
					ImagePullPolicy: corev1.PullIfNotPresent,
					Command: []string{
						"sh",
						"-c",
						fmt.Sprintf(
							"if wget -T 5 -qO- --header='name: network-policy-test' http://%s/hello | grep -q 'Hello network-policy-test'; then echo ACCESS_GRANTED; else echo ACCESS_DENIED; fi; sleep 3600",
							serviceName,
						),
					},
				},
			},
		},
	}
}

func PodPhase(t *testing.T, ctx context.Context, namespace, name string) func() corev1.PodPhase {
	return func() corev1.PodPhase {
		pod, err := TestClient(t).CoreV1().Pods(namespace).Get(ctx, name, metav1.GetOptions{})
		if err != nil {
			if !k8serrors.IsNotFound(err) {
				t.Logf("failed to read pod %s/%s: %v", namespace, name, err)
			}
			return ""
		}
		return pod.Status.Phase
	}
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
