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
	"testing"

	"io/ioutil"

	. "github.com/apache/camel-k/v2/e2e/support"
	. "github.com/onsi/gomega"
	"github.com/stretchr/testify/require"
)

/*
	go test -v -tags=integration ./e2e/knative --run TestOpenAPIService
*/

func TestOpenAPIService(t *testing.T) {
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		openapiContent, err := ioutil.ReadFile("./files/petstore-api.yaml")
		require.NoError(t, err)
		var cmDataProps = make(map[string]string)
		cmDataProps["petstore-api.yaml"] = string(openapiContent)
		CreatePlainTextConfigmap(t, ctx, ns, "my-openapi-knative", cmDataProps)

		g.Expect(KamelRun(t, ctx, ns, "--name", "petstore", "--open-api", "configmap:my-openapi-knative", "files/petstore.yaml").Execute()).To(Succeed())

		g.Eventually(KnativeService(t, ctx, ns, "petstore"), TestTimeoutLong).
			Should(Not(BeNil()))
		g.Eventually(IntegrationLogs(t, ctx, ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started listPets (rest://get:/v1:/pets)"))
		g.Eventually(IntegrationLogs(t, ctx, ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started createPets (rest://post:/v1:/pets)"))
		g.Eventually(IntegrationLogs(t, ctx, ns, "petstore"), TestTimeoutMedium).
			Should(ContainSubstring("Started showPetById (rest://get:/v1:/pets/%7BpetId%7D)"))
	})
}
