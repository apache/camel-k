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
	"testing"
	"time"

	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	. "github.com/apache/camel-k/v2/e2e/support"
	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
)

func TestMasterTrait(t *testing.T) {
	t.Parallel()
	WithNewTestNamespace(t, func(ctx context.Context, g *WithT, ns string) {
		t.Run("master works with properties", func(t *testing.T) {
			name := "master"
			// Create RBAC resources for the master component
			CreateMasterRBAC(t, ctx, g, ns, name, "default")

			// Run using Quarkus properties instead of deprecated trait
			g.Expect(KamelRun(t, ctx, ns, "files/Master.java",
				"-p", fmt.Sprintf("quarkus.camel.cluster.kubernetes.resource-name=%s-lock", name),
				"-p", "quarkus.camel.cluster.kubernetes.resource-type=Lease",
				"-p", fmt.Sprintf("quarkus.camel.cluster.kubernetes.labels.\"camel.apache.org/integration\"=%s", name),
			).Execute()).To(Succeed())
			g.Eventually(IntegrationPodPhase(t, ctx, ns, name), TestTimeoutLong).Should(Equal(corev1.PodRunning))
			g.Eventually(IntegrationLogs(t, ctx, ns, name), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))
			g.Expect(Kamel(t, ctx, "delete", "--all", "-n", ns).Execute()).To(Succeed())
		})

		t.Run("only one integration with master runs using properties", func(t *testing.T) {
			nameFirst := RandomizedSuffixName("first")
			nameSecond := RandomizedSuffixName("second")
			lockName := nameFirst + "-lock"

			CreateMasterRBAC(t, ctx, g, ns, nameFirst, "default")
			CreateMasterRBAC(t, ctx, g, ns, nameSecond, "default")

			g.Expect(KamelRun(t, ctx, ns, "files/Master.java", "--name", nameFirst,
				"--label", "leader-group=same",
				"-t", "owner.target-labels=leader-group",
				"-p", fmt.Sprintf("quarkus.camel.cluster.kubernetes.resource-name=%s", lockName),
				"-p", "quarkus.camel.cluster.kubernetes.resource-type=Lease",
				"-p", "quarkus.camel.cluster.kubernetes.labels.\"leader-group\"=same",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameFirst, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameFirst), TestTimeoutShort).Should(ContainSubstring("Magicstring!"))

			g.Expect(KamelRun(t, ctx, ns, "files/Master.java", "--name", nameSecond,
				"--label", "leader-group=same",
				"-t", "owner.target-labels=leader-group",
				"-p", fmt.Sprintf("quarkus.camel.cluster.kubernetes.resource-name=%s", lockName),
				"-p", "quarkus.camel.cluster.kubernetes.resource-type=Lease",
				"-p", "quarkus.camel.cluster.kubernetes.labels.\"leader-group\"=same",
			).Execute()).To(Succeed())
			g.Eventually(IntegrationLogs(t, ctx, ns, nameSecond), TestTimeoutShort).Should(ContainSubstring("started in"))
			g.Eventually(IntegrationLogs(t, ctx, ns, nameSecond), 15*time.Second).ShouldNot(ContainSubstring("Magicstring!"))
			g.Eventually(IntegrationConditionStatus(t, ctx, ns, nameSecond, v1.IntegrationConditionReady), TestTimeoutShort).Should(Equal(corev1.ConditionTrue))
		})
	})
}

// CreateMasterRBAC creates the Role and RoleBinding
func CreateMasterRBAC(t *testing.T, ctx context.Context, g *WithT, ns string, name string, serviceAccount string) {
	t.Helper()

	role := &rbacv1.Role{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-master",
			Namespace: ns,
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"coordination.k8s.io"},
				Resources: []string{"leases"},
				Verbs:     []string{"create", "delete", "deletecollection", "get", "list", "patch", "update", "watch"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"pods"},
				Verbs:     []string{"get", "list", "watch"},
			},
		},
	}
	g.Expect(TestClient(t).Create(ctx, role)).To(Succeed())

	roleBinding := &rbacv1.RoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name + "-master",
			Namespace: ns,
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      serviceAccount,
				Namespace: ns,
			},
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     name + "-master",
		},
	}
	g.Expect(TestClient(t).Create(ctx, roleBinding)).To(Succeed())
}
