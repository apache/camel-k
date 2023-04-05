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
	"testing"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	"github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestErroredTrait(t *testing.T) {
	RegisterTestingT(t)

	t.Run("Integration trait should fail", func(t *testing.T) {
		name := "it-errored"
		Expect(KamelRunWithID(operatorID, ns, "files/Java.java",
			"--name", name,
			"-t", "kamelets.list=missing",
		).Execute()).To(Succeed())
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(And(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionInitializationFailedReason)),
			WithTransform(IntegrationConditionMessage, HavePrefix("error during trait customization")),
		))
	})

	t.Run("Pipe trait should fail", func(t *testing.T) {
		name := "kb-errored"
		Expect(KamelBindWithID(operatorID, ns, "timer:foo", "log:bar",
			"--name", name,
			"-t", "kamelets.list=missing",
		).Execute()).To(Succeed())
		// Pipe
		Eventually(PipePhase(ns, name), TestTimeoutShort).Should(Equal(v1.PipePhaseError))
		Eventually(PipeConditionStatus(ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		Eventually(PipeCondition(ns, name, v1.PipeConditionReady), TestTimeoutShort).Should(
			WithTransform(PipeConditionMessage, And(
				ContainSubstring("error during trait customization"),
				ContainSubstring("[missing] not found"),
			)))
		// Integration related
		Eventually(IntegrationPhase(ns, name), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationConditionStatus(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(ns, name, v1.IntegrationConditionReady), TestTimeoutShort).Should(And(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionInitializationFailedReason)),
			WithTransform(IntegrationConditionMessage, HavePrefix("error during trait customization")),
		))
	})

	// Clean up
	Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
}
