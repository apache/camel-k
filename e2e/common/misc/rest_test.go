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

package misc

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"testing"
	"time"

	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

func TestRunRest(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		ocp, err := openshift.IsOpenShift(TestClient(t))
		require.NoError(t, err)

		g.Expect(KamelRun(t, ctx, ns, "files/rest-consumer.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "rest-consumer"), TestTimeoutLong).Should(Equal(corev1.PodRunning))

		t.Run("Service works", func(t *testing.T) {
			name := RandomizedSuffixName("John")
			service := Service(t, ctx, ns, "rest-consumer")
			g.Eventually(service, TestTimeoutShort).ShouldNot(BeNil())
			g.Expect(KamelRun(t, ctx, ns, "files/rest-producer.yaml", "-p", "serviceName=rest-consumer", "-p", "name="+name).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, "rest-producer"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, "rest-consumer"), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("get %s", name)))
			g.Eventually(IntegrationLogs(t, ctx, ns, "rest-producer"), TestTimeoutLong).Should(ContainSubstring(fmt.Sprintf("%s Doe", name)))
		})

		if ocp {
			t.Run("Route works", func(t *testing.T) {
				name := RandomizedSuffixName("Peter")
				route := Route(t, ctx, ns, "rest-consumer")
				g.Eventually(route, TestTimeoutShort).ShouldNot(BeNil())
				g.Eventually(RouteStatus(t, ctx, ns, "rest-consumer"), TestTimeoutMedium).Should(Equal("True"))
				url := fmt.Sprintf("http://%s/customers/%s", route().Spec.Host, name)
				g.Eventually(httpRequest(url), TestTimeoutMedium).Should(Equal(fmt.Sprintf("%s Doe", name)))
				g.Eventually(IntegrationLogs(t, ctx, ns, "rest-consumer"), TestTimeoutShort).Should(ContainSubstring(fmt.Sprintf("get %s", name)))
			})
		}
	})
}

func httpRequest(url string) func() (string, error) {
	return func() (string, error) {
		client := &http.Client{Timeout: 3 * time.Second}
		response, err := client.Get(url)
		if err != nil {
			return "", err
		}
		defer response.Body.Close()

		body, err := io.ReadAll(response.Body)
		if err != nil {
			return "", err
		}

		return string(body), nil
	}
}
