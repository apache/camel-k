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

package common

import (
	"context"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestInitContainerTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Init container write a file", func(t *testing.T) {
			name := RandomizedSuffixName("init")
			g.Expect(KamelRun(t, ctx, ns,
				"files/init-container.yaml",
				"-t",
				"mount.empty-dirs=common:/tmp",
				"-t",
				"init-containers.init-tasks=init;alpine;/bin/sh -c \"echo helloInit >> /tmp/init\"",
				"--name",
				name).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("helloInit"))
		})
		t.Run("Sidecar container write a file", func(t *testing.T) {
			name := RandomizedSuffixName("sidecar")
			g.Expect(KamelRun(t, ctx, ns,
				"files/init-container.yaml",
				"-t",
				"mount.empty-dirs=common:/tmp",
				"-t",
				"init-containers.sidecar-tasks=sidecar;alpine;/bin/sh -c \"for i in $(seq 1 10); do echo helloSidecar$i > /tmp/sidecar_$i.txt; sleep 1; done\"",
				"--name",
				name).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady)).
				Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("helloSidecar10"))
		})
	})
}
