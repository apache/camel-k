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

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestSyntheticIntegrationOff(t *testing.T) {
	RegisterTestingT(t)
	WithNewTestNamespace(t, func(ns string) {
		// Install Camel K without synthetic Integration feature variable (default)
		operatorID := "camel-k-synthetic-env-off"
		Expect(KamelInstallWithID(operatorID, ns).Execute()).To(Succeed())

		// Run the external deployment
		ExpectExecSucceed(t, Kubectl("apply", "-f", "files/deploy.yaml", "-n", ns))
		Eventually(DeploymentCondition(ns, "my-camel-sb-svc", appsv1.DeploymentProgressing), TestTimeoutShort).
			Should(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal("NewReplicaSetAvailable"),
			}))

		// Label the deployment --> Verify the Integration is not created
		ExpectExecSucceed(t, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		Eventually(Integration(ns, "my-it"), TestTimeoutShort).Should(BeNil())
	})
}
func TestSyntheticIntegrationFromDeployment(t *testing.T) {
	RegisterTestingT(t)
	WithNewTestNamespace(t, func(ns string) {
		// Install Camel K with the synthetic Integration feature variable
		operatorID := "camel-k-synthetic-env"
		Expect(KamelInstallWithID(operatorID, ns,
			"--operator-env-vars", "CAMEL_K_SYNTHETIC_INTEGRATIONS=true",
		).Execute()).To(Succeed())

		// Run the external deployment
		ExpectExecSucceed(t, Kubectl("apply", "-f", "files/deploy.yaml", "-n", ns))
		Eventually(DeploymentCondition(ns, "my-camel-sb-svc", appsv1.DeploymentProgressing), TestTimeoutShort).
			Should(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal("NewReplicaSetAvailable"),
			}))

		// Label the deployment --> Verify the Integration is created (cannot still monitor)
		ExpectExecSucceed(t, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		Eventually(IntegrationPhase(ns, "my-it"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
		Eventually(IntegrationConditionStatus(ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		Eventually(IntegrationCondition(ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionMonitoringPodsAvailableReason)))

		// Label the deployment template --> Verify the Integration is monitored
		ExpectExecSucceed(t, Kubectl("patch", "deployment", "my-camel-sb-svc", "--patch", `{"spec": {"template": {"metadata": {"labels": {"camel.apache.org/integration": "my-it"}}}}}`, "-n", ns))
		Eventually(IntegrationPhase(ns, "my-it"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
		Eventually(IntegrationConditionStatus(ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		one := int32(1)
		Eventually(IntegrationStatusReplicas(ns, "my-it"), TestTimeoutShort).Should(Equal(&one))

		// Delete the deployment --> Verify the Integration is eventually garbage collected
		ExpectExecSucceed(t, Kubectl("delete", "deploy", "my-camel-sb-svc", "-n", ns))
		Eventually(Integration(ns, "my-it"), TestTimeoutShort).Should(BeNil())

		// Recreate the deployment and label --> Verify the Integration is monitored
		ExpectExecSucceed(t, Kubectl("apply", "-f", "files/deploy.yaml", "-n", ns))
		ExpectExecSucceed(t, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		ExpectExecSucceed(t, Kubectl("patch", "deployment", "my-camel-sb-svc", "--patch", `{"spec": {"template": {"metadata": {"labels": {"camel.apache.org/integration": "my-it"}}}}}`, "-n", ns))
		Eventually(IntegrationPhase(ns, "my-it"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
		Eventually(IntegrationConditionStatus(ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationStatusReplicas(ns, "my-it"), TestTimeoutShort).Should(Equal(&one))

		// Remove label from the deployment --> Verify the Integration is deleted
		ExpectExecSucceed(t, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration-", "-n", ns))
		Eventually(Integration(ns, "my-it"), TestTimeoutShort).Should(BeNil())

		// Add label back to the deployment --> Verify the Integration is created
		ExpectExecSucceed(t, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		Eventually(IntegrationPhase(ns, "my-it"), TestTimeoutShort).Should(Equal(v1.IntegrationPhaseRunning))
		Eventually(IntegrationConditionStatus(ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		Eventually(IntegrationStatusReplicas(ns, "my-it"), TestTimeoutShort).Should(Equal(&one))
		// Scale the deployment --> verify replicas are correctly set
		ExpectExecSucceed(t, Kubectl("scale", "deploy", "my-camel-sb-svc", "--replicas", "2", "-n", ns))
		two := int32(2)
		Eventually(IntegrationStatusReplicas(ns, "my-it"), TestTimeoutShort).Should(Equal(&two))

		// Delete Integration and deployments --> verify no Integration exists any longer
		Expect(Kamel("delete", "--all", "-n", ns).Execute()).To(Succeed())
		ExpectExecSucceed(t, Kubectl("delete", "deploy", "my-camel-sb-svc", "-n", ns))
		Eventually(Integration(ns, "my-it"), TestTimeoutShort).Should(BeNil())
	})
}
