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
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestMountTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		expectLog := func(name, expected string) {
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady),
				TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring(expected))
		}

		t.Run("Mount configs from configmap", func(t *testing.T) {
			require.NoError(t, CreatePlainTextConfigmap(t, ctx, ns, "mount-cm-config",
				map[string]string{"app.properties": "my.message=configmap-config-value"}))
			name := RandomizedSuffixName("mount-cm-config")
			g.Expect(KamelRun(t, ctx, ns, "./files/mount/config-route.yaml",
				"--name", name,
				"-t", "mount.configs=configmap:mount-cm-config",
			).Execute()).To(Succeed())
			expectLog(name, "config property is: configmap-config-value")
		})

		t.Run("Mount configs from secret", func(t *testing.T) {
			require.NoError(t, CreatePlainTextSecret(t, ctx, ns, "mount-sec-config",
				map[string]string{"app.properties": "my.message=secret-config-value"}))
			name := RandomizedSuffixName("mount-sec-config")
			g.Expect(KamelRun(t, ctx, ns, "./files/mount/config-route.yaml",
				"--name", name,
				"-t", "mount.configs=secret:mount-sec-config",
			).Execute()).To(Succeed())
			expectLog(name, "config property is: secret-config-value")
		})

		t.Run("Mount resources with filtered key and destination", func(t *testing.T) {
			require.NoError(t, CreatePlainTextConfigmap(t, ctx, ns, "mount-cm-resource",
				map[string]string{
					"my-resource.txt": "configmap-resource-value",
					"other.txt":       "should-not-be-mounted",
				}))
			name := RandomizedSuffixName("mount-cm-resource")
			g.Expect(KamelRun(t, ctx, ns, "./files/mount/resource-route.yaml",
				"--name", name,
				"-t", "mount.resources=configmap:mount-cm-resource/my-resource.txt@/tmp/mount/my-resource.txt",
			).Execute()).To(Succeed())
			expectLog(name, "resource content is: configmap-resource-value")
		})

		t.Run("Mount empty-dirs", func(t *testing.T) {
			name := RandomizedSuffixName("mount-emptydir")
			g.Expect(KamelRun(t, ctx, ns, "./files/mount/volume-route.yaml",
				"--name", name,
				"-t", "mount.empty-dirs=mount-scratch:/tmp/scratch:100Mi",
				"-p", "volume.path=/tmp/scratch",
				"-p", "volume.message=emptydir-value",
			).Execute()).To(Succeed())
			expectLog(name, "volume content is: emptydir-value")
		})

		t.Run("Mount volumes with dynamic PVC", func(t *testing.T) {
			name := RandomizedSuffixName("mount-pvc")
			g.Expect(KamelRun(t, ctx, ns, "./files/mount/volume-route.yaml",
				"--name", name,
				"-t", "mount.volumes=mount-pvc:/tmp/pvc:100Mi:ReadWriteOnce",
				"-p", "volume.path=/tmp/pvc",
				"-p", "volume.message=pvc-value",
			).Execute()).To(Succeed())
			expectLog(name, "volume content is: pvc-value")
		})
	})
}
