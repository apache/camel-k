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

package install

import (
	"context"
	"fmt"

	rbacv1 "k8s.io/api/rbac/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	rbacv1ac "k8s.io/client-go/applyconfigurations/rbac/v1"
	"k8s.io/client-go/kubernetes"

	"github.com/apache/camel-k/pkg/util/knative"
)

const knativeAddressableResolverClusterRoleName = "addressable-resolver"

// BindKnativeAddressableResolverClusterRole binds the Knative addressable resolver aggregated ClusterRole
// to the operator ServiceAccount.
func BindKnativeAddressableResolverClusterRole(ctx context.Context, c kubernetes.Interface, namespace string, operatorNamespace string) error {
	if isKnative, err := knative.IsInstalled(ctx, c); err != nil {
		return err
	} else if !isKnative {
		return nil
	}
	if namespace != "" {
		return applyAddressableResolverRoleBinding(ctx, c, namespace, operatorNamespace)
	}
	return applyAddressableResolverClusterRoleBinding(ctx, c, operatorNamespace)
}

func applyAddressableResolverRoleBinding(ctx context.Context, c kubernetes.Interface, namespace string, operatorNamespace string) error {
	rb := rbacv1ac.RoleBinding(fmt.Sprintf("%s-addressable-resolver", serviceAccountName), namespace).
		WithSubjects(
			rbacv1ac.Subject().
				WithKind("ServiceAccount").
				WithNamespace(operatorNamespace).
				WithName(serviceAccountName),
		).
		WithRoleRef(rbacv1ac.RoleRef().
			WithAPIGroup(rbacv1.GroupName).
			WithKind("ClusterRole").
			WithName(knativeAddressableResolverClusterRoleName))

	rb.WithLabels(map[string]string{"app": "camel-k"})
	_, err := c.RbacV1().RoleBindings(namespace).
		Apply(ctx, rb, metav1.ApplyOptions{FieldManager: serviceAccountName, Force: true})

	return err
}

func applyAddressableResolverClusterRoleBinding(ctx context.Context, c kubernetes.Interface, operatorNamespace string) error {
	crb := rbacv1ac.ClusterRoleBinding(fmt.Sprintf("%s-addressable-resolver", serviceAccountName)).
		WithSubjects(
			rbacv1ac.Subject().
				WithKind("ServiceAccount").
				WithNamespace(operatorNamespace).
				WithName(serviceAccountName),
		).
		WithRoleRef(rbacv1ac.RoleRef().
			WithAPIGroup(rbacv1.GroupName).
			WithKind("ClusterRole").
			WithName(knativeAddressableResolverClusterRoleName))

	crb.WithLabels(map[string]string{"app": "camel-k"})
	_, err := c.RbacV1().ClusterRoleBindings().
		Apply(ctx, crb, metav1.ApplyOptions{FieldManager: serviceAccountName, Force: true})

	return err
}
