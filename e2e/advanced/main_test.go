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

package advanced

import (
	"fmt"
	"os"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"

	corev1 "k8s.io/api/core/v1"
)

func TestMain(m *testing.M) {
	justCompile := GetEnvOrDefault("CAMEL_K_E2E_JUST_COMPILE", "false")
	if justCompile == "true" {
		os.Exit(m.Run())
	}

	fastSetup := GetEnvOrDefault("CAMEL_K_E2E_FAST_SETUP", "false")
	if fastSetup == "true" {
		operatorID := platform.DefaultPlatformName
		ns := GetEnvOrDefault("CAMEL_K_GLOBAL_OPERATOR_NS", TestDefaultNamespace)

		g := NewGomega(func(message string, callerSkip ...int) {
			fmt.Printf("Test fast setup failed! - %s\n", message)
		})

		var t *testing.T
		g.Expect(TestClient(t)).ShouldNot(BeNil())
		ctx := TestContext()
		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/Java.java").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "java"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "java", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, "java"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/yaml.yaml").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "yaml"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "yaml", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, "yaml"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

		g.Expect(CamelKRunWithID(t, ctx, operatorID, ns, "files/timer-source.groovy").Execute()).To(Succeed())
		g.Eventually(IntegrationPodPhase(t, ctx, ns, "timer-source"), TestTimeoutLong).Should(Equal(corev1.PodRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "timer-source", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationLogs(t, ctx, ns, "timer-source"), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
	}

	os.Exit(m.Run())
}
