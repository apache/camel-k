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

package config

import (
	"context"
	"strconv"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestConfigmapHotReload(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("config-configmap-route")

		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my configmap content"
		CreatePlainTextConfigmapWithLabels(t, ctx, ns, "my-hot-cm", cmData, map[string]string{"camel.apache.org/integration": "test"})

		g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-route.yaml",
			"--config", "configmap:my-hot-cm",
			"-t", "mount.hot-reload=true",
			"--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("my configmap content"))

		cmData["my-configmap-key"] = "my configmap content updated"
		UpdatePlainTextConfigmapWithLabels(t, ctx, ns, "my-hot-cm", cmData, map[string]string{"camel.apache.org/integration": "test"})
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("my configmap content updated"))
	})
}

func TestConfigmapHotReloadDefault(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("config-configmap-route")

		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my configmap content"
		CreatePlainTextConfigmapWithLabels(t, ctx, ns, "my-hot-cm-2", cmData, map[string]string{"camel.apache.org/integration": "test"})

		g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-route.yaml",
			"--config", "configmap:my-hot-cm-2",
			"--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("my configmap content"))

		cmData["my-configmap-key"] = "my configmap content updated"
		UpdatePlainTextConfigmapWithLabels(t, ctx, ns, "my-hot-cm-2", cmData, map[string]string{"camel.apache.org/integration": "test"})
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(Not(ContainSubstring("my configmap content updated")))
	})
}

func TestSecretHotReload(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("config-secret-route")

		var secData = make(map[string]string)
		secData["my-secret-key"] = "very top secret"
		CreatePlainTextSecretWithLabels(t, ctx, ns, "my-hot-sec", secData, map[string]string{"camel.apache.org/integration": "test"})

		g.Expect(KamelRun(t, ctx, ns, "./files/config-secret-route.yaml",
			"--config", "secret:my-hot-sec",
			"-t", "mount.hot-reload=true",
			"--name", name).Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("very top secret"))

		secData["my-secret-key"] = "very top secret updated"
		UpdatePlainTextSecretWithLabels(t, ctx, ns, "my-hot-sec", secData, map[string]string{"camel.apache.org/integration": "test"})
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("very top secret updated"))
	})
}

func TestConfigmapWithOwnerRefHotReloadDefault(t *testing.T) {
	t.Parallel()
	CheckConfigmapWithOwnerRef(t, false)
}

func TestConfigmapWithOwnerRefHotReload(t *testing.T) {
	t.Parallel()
	CheckConfigmapWithOwnerRef(t, true)
}

func CheckConfigmapWithOwnerRef(t *testing.T, hotreload bool) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("config-configmap-route")
		cmName := RandomizedSuffixName("my-hot-cm-")
		g.Expect(KamelRun(t, ctx, ns, "./files/config-configmap-route.yaml",
			"--config", "configmap:"+cmName,
			"--name", name,
			"-t", "mount.hot-reload="+strconv.FormatBool(hotreload)).Execute()).To(Succeed())

		g.Eventually(IntegrationPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(v1.IntegrationPhaseError))
		var cmData = make(map[string]string)
		cmData["my-configmap-key"] = "my configmap content"
		CreatePlainTextConfigmapWithOwnerRefWithLabels(t, ctx, ns, cmName, cmData, name, Integration(t, ctx, ns, name)().UID, map[string]string{"camel.apache.org/integration": "test"})
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutLong).Should(ContainSubstring("my configmap content"))
		cmData["my-configmap-key"] = "my configmap content updated"
		UpdatePlainTextConfigmapWithLabels(t, ctx, ns, cmName, cmData, map[string]string{"camel.apache.org/integration": "test"})
		if hotreload {
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutLong).Should(ContainSubstring("my configmap content updated"))
		} else {
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutLong).Should(Not(ContainSubstring("my configmap content updated")))
		}
	})
}
