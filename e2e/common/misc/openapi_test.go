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
	"fmt"
	"os"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestOpenAPIContractFirst(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		name := RandomizedSuffixName("petstore")
		openapiContent, err := os.ReadFile("./files/petstore-api.yaml")
		require.NoError(t, err)
		var cmDataProps = make(map[string]string)
		cmDataProps["petstore-api.yaml"] = string(openapiContent)
		CreatePlainTextConfigmap(t, ctx, ns, "my-openapi", cmDataProps)

		g.Expect(KamelRun(t, ctx, ns,
			"--name", name, "--resource", "configmap:my-openapi", "files/petstore.yaml").
			Execute()).To(Succeed())

		g.Eventually(IntegrationConditionStatus(t, ctx, ns, name, v1.IntegrationConditionReady), TestTimeoutMedium).
			Should(Equal(corev1.ConditionTrue))
		g.Eventually(Service(t, ctx, ns, name), TestTimeoutShort).ShouldNot(BeNil())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, name)).Should(Equal(corev1.PodRunning))
		// Let's make sure the Integration is ready to receive traffic
		g.Eventually(IntegrationLogs(t, ctx, ns, name)).Should(ContainSubstring("Listening on: http://0.0.0.0:8080"))
		pod := IntegrationPod(t, ctx, ns, name)()
		g.Expect(pod).NotTo(BeNil())
		response, err := TestClient(t).CoreV1().RESTClient().Get().
			Timeout(30 * time.Second).
			AbsPath(fmt.Sprintf("/api/v1/namespaces/%s/pods/%s/proxy/v1/pets", pod.Namespace, pod.Name)).
			DoRaw(ctx)
		require.NoError(t, err)
		assert.Equal(t, "listPets", string(response))
	})
}
