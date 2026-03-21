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

package gateway

import (
	"context"
	"os/exec"
	"testing"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
)

func TestGatewayTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		g.Expect(KamelRun(t, ctx, ns, "files/PlatformHttpServer.java",
			"-t", "gateway.enabled=true",
			"-t", "gateway.class-name=envoy",
		).Execute()).To(Succeed())
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "platform-http-server",
			v1.IntegrationConditionReady), TestTimeoutMedium).Should(Equal(corev1.ConditionTrue))
		g.Eventually(Gateway(t, ctx, ns, "platform-http-server"), TestTimeoutShort).Should(Not(BeNil()))
		// Wait for the address to be assigned
		var gwAddress string

		// IMPORTANT NOTE: this test would likely fail if the Envoy gateway is not able
		// to assign an address correctly. In our case we need to make sure to run
		// `minikube tunnel` before running this test. It requires sudo.

		g.Eventually(func() string {
			gw := Gateway(t, ctx, ns, "platform-http-server")()
			if gw == nil || len(gw.Status.Addresses) == 0 {
				return ""
			}
			gwAddress = string(gw.Status.Addresses[0].Value)

			return gwAddress
		}, TestTimeoutShort).ShouldNot(BeEmpty(), "expected gateway to have an assigned address")

		g.Eventually(func() (string, error) {
			cmd := exec.Command("curl",
				"-s",
				"-H", "name: test!",
				"http://"+gwAddress+":8080/hello",
			)
			out, err := cmd.CombinedOutput()

			return string(out), err
		}, TestTimeoutMedium).Should(ContainSubstring("Hello test!"))
	})
}
