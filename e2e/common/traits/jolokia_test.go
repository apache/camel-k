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

package traits

import (
	"context"
	"fmt"
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestJolokiaTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("Run Java with Jolokia", func(t *testing.T) {
			name := RandomizedSuffixName("java")
			g.Expect(KamelRun(t, ctx, ns, "files/Java.java", "--name", name,
				"-t", "jolokia.enabled=true",
				"-t", "jolokia.use-ssl-client-authentication=false",
				"-t", "jolokia.protocol=http",
				"-t", "jolokia.extended-client-check=false",
				// TODO check if the WA is valid
				"--build-property", "jolokia=true",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			pod := IntegrationPod(t, ctx, ns, name)
			response, err := TestClient(t).CoreV1().RESTClient().Get().
				AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/jolokia/", ns, pod().Name)).DoRaw(ctx)
			g.Expect(err).To(BeNil())
			g.Expect(response).To(ContainSubstring(`"status":200`))
		})
	})
}
