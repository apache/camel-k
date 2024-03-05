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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

const customLabel = "custom-label"

func TestBundleKameletUpdate(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(g *WithT, ns string) {
		operatorID := "camel-k-kamelet-bundle"
		g.Expect(CopyCamelCatalog(t, ns, operatorID)).To(Succeed())
		g.Expect(CopyIntegrationKits(t, ns, operatorID)).To(Succeed())
		g.Expect(KamelInstallWithID(t, operatorID, ns)).To(Succeed())

		g.Eventually(SelectedPlatformPhase(t, ns, operatorID), TestTimeoutMedium).Should(Equal(v1.IntegrationPlatformPhaseReady))

		g.Expect(createBundleKamelet(t, operatorID, ns, "my-http-sink")()).To(Succeed()) // Going to be replaced
		g.Expect(createUserKamelet(t, operatorID, ns, "user-sink")()).To(Succeed())      // Left intact by the operator

		g.Eventually(Kamelet(t, "my-http-sink", ns)).
			Should(WithTransform(KameletLabels, HaveKeyWithValue(customLabel, "true")))
		g.Consistently(Kamelet(t, "user-sink", ns), 5*time.Second, 1*time.Second).
			Should(WithTransform(KameletLabels, HaveKeyWithValue(customLabel, "true")))

		g.Expect(Kamel(t, "delete", "--all", "-n", ns).Execute()).To(Succeed())
	})
}

func createBundleKamelet(t *testing.T, operatorID string, ns string, name string) func() error {
	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "kamelet:source",
		},
	}

	labels := map[string]string{
		customLabel:            "true",
		v1.KameletBundledLabel: "true",
	}
	return CreateKamelet(t, operatorID, ns, name, flow, nil, labels)
}

func createUserKamelet(t *testing.T, operatorID string, ns string, name string) func() error {
	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": "kamelet:source",
		},
	}

	labels := map[string]string{
		customLabel: "true",
	}
	return CreateKamelet(t, operatorID, ns, name, flow, nil, labels)
}
