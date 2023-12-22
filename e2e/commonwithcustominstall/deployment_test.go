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

package commonwithcustominstall

import (
	"testing"

	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"os/exec"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestDeploymentFailureShouldReportIntegrationCondition(t *testing.T) {
	WithNewTestNamespace(t, func(ns string) {
		op := "camel-k-failing-deploy"
		nsRestr := "restr"
		Expect(KamelInstallWithID(op, ns, "--global", "--force").Execute()).To(Succeed())
		// Create restricted namespace
		ExpectExecSucceed(t,
			exec.Command(
				"kubectl",
				"create",
				"ns",
				nsRestr,
			),
		)
		ExpectExecSucceed(t,
			exec.Command(
				"kubectl",
				"label",
				"--overwrite",
				"ns",
				nsRestr,
				"pod-security.kubernetes.io/enforce=baseline",
				"pod-security.kubernetes.io/enforce-version=latest",
				"pod-security.kubernetes.io/enforce=restricted",
				"pod-security.kubernetes.io/warn-version=latest",
				"pod-security.kubernetes.io/audit=restricted",
				"pod-security.kubernetes.io/audit-version=latest",
			),
		)
		// Create an Integration into a restricted namespace
		name := RandomizedSuffixName("java-fail")
		Expect(KamelRunWithID(op, ns, "files/Java.java", "--name", name, "-n", nsRestr).Execute()).To(Succeed())
		// Check the error is reported into the Integration
		Eventually(IntegrationPhase(nsRestr, name), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseError))
		Eventually(IntegrationCondition(nsRestr, name, v1.IntegrationConditionReady)().Status).
			Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(nsRestr, name, v1.IntegrationConditionReady)().Message).
			Should(ContainSubstring("is forbidden: violates PodSecurity"))
		// Clean up
		Eventually(DeleteIntegrations(nsRestr)).Should(Equal(0))
	})
}
