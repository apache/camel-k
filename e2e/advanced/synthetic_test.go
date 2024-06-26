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
	"context"
	"testing"

	. "github.com/onsi/gomega"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/envvar"
	. "github.com/onsi/gomega/gstruct"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
)

func TestSyntheticIntegrationOff(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Install Camel K without synthetic Integration feature variable (default)
		InstallOperator(t, g, ns)

		// Run the external deployment
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/deploy.yaml", "-n", ns))
		g.Eventually(DeploymentCondition(t, ctx, ns, "my-camel-sb-svc", appsv1.DeploymentProgressing), TestTimeoutShort).
			Should(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal("NewReplicaSetAvailable"),
			}))

		// Label the deployment --> Verify the Integration is not created
		ExpectExecSucceed(t, g, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		g.Eventually(Integration(t, ctx, ns, "my-it"), TestTimeoutShort).Should(BeNil())
	})
}
func TestSyntheticIntegrationFromDeployment(t *testing.T) {
	t.Parallel()

	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		// Install Camel K with the synthetic Integration feature variable
		// g.Expect(InstallOperator(t, ctx, operatorID, ns,
		// 	"--operator-env-vars", "CAMEL_K_SYNTHETIC_INTEGRATIONS=true",
		// )).To(Succeed())
		InstallOperator(t, g, ns)
		g.Eventually(OperatorPodHas(t, ctx, ns, func(op *corev1.Pod) bool {
			if envVar := envvar.Get(op.Spec.Containers[0].Env, "CAMEL_K_SYNTHETIC_INTEGRATIONS"); envVar != nil {
				return envVar.Value == "true"
			}
			return false

		}), TestTimeoutShort).Should(BeTrue())

		// Run the external deployment
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/deploy.yaml", "-n", ns))
		g.Eventually(DeploymentCondition(t, ctx, ns, "my-camel-sb-svc", appsv1.DeploymentProgressing), TestTimeoutShort).
			Should(MatchFields(IgnoreExtras, Fields{
				"Status": Equal(corev1.ConditionTrue),
				"Reason": Equal("NewReplicaSetAvailable"),
			}))

		// Label the deployment --> Verify the Integration is created (cannot still monitor)
		ExpectExecSucceed(t, g, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		g.Eventually(IntegrationPhase(t, ctx, ns, "my-it"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionFalse))
		g.Eventually(IntegrationCondition(t, ctx, ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(
			WithTransform(IntegrationConditionReason, Equal(v1.IntegrationConditionMonitoringPodsAvailableReason)))

		// Label the deployment template --> Verify the Integration is monitored
		ExpectExecSucceed(t, g, Kubectl("patch", "deployment", "my-camel-sb-svc", "--patch", `{"spec": {"template": {"metadata": {"labels": {"camel.apache.org/integration": "my-it"}}}}}`, "-n", ns))
		g.Eventually(IntegrationPhase(t, ctx, ns, "my-it"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		one := int32(1)
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "my-it"), TestTimeoutShort).Should(Equal(&one))

		// Delete the deployment --> Verify the Integration is eventually garbage collected
		ExpectExecSucceed(t, g, Kubectl("delete", "deploy", "my-camel-sb-svc", "-n", ns))
		g.Eventually(Integration(t, ctx, ns, "my-it"), TestTimeoutShort).Should(BeNil())

		// Recreate the deployment and label --> Verify the Integration is monitored
		ExpectExecSucceed(t, g, Kubectl("apply", "-f", "files/deploy.yaml", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		ExpectExecSucceed(t, g, Kubectl("patch", "deployment", "my-camel-sb-svc", "--patch", `{"spec": {"template": {"metadata": {"labels": {"camel.apache.org/integration": "my-it"}}}}}`, "-n", ns))
		g.Eventually(IntegrationPhase(t, ctx, ns, "my-it"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "my-it"), TestTimeoutShort).Should(Equal(&one))

		// Remove label from the deployment --> Verify the Integration is deleted
		ExpectExecSucceed(t, g, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration-", "-n", ns))
		g.Eventually(Integration(t, ctx, ns, "my-it"), TestTimeoutShort).Should(BeNil())

		// Add label back to the deployment --> Verify the Integration is created
		ExpectExecSucceed(t, g, Kubectl("label", "deploy", "my-camel-sb-svc", "camel.apache.org/integration=my-it", "-n", ns))
		g.Eventually(IntegrationPhase(t, ctx, ns, "my-it"), TestTimeoutMedium).Should(Equal(v1.IntegrationPhaseRunning))
		g.Eventually(IntegrationConditionStatus(t, ctx, ns, "my-it", v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "my-it"), TestTimeoutShort).Should(Equal(&one))
		// Scale the deployment --> verify replicas are correctly set
		ExpectExecSucceed(t, g, Kubectl("scale", "deploy", "my-camel-sb-svc", "--replicas", "2", "-n", ns))
		two := int32(2)
		g.Eventually(IntegrationStatusReplicas(t, ctx, ns, "my-it"), TestTimeoutShort).Should(Equal(&two))

		// Delete Integration and deployments --> verify no Integration exists any longer
		g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		ExpectExecSucceed(t, g, Kubectl("delete", "deploy", "my-camel-sb-svc", "-n", ns))
		g.Eventually(Integration(t, ctx, ns, "my-it"), TestTimeoutShort).Should(BeNil())
	})
}
