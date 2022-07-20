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

package kubernetes

import (
	"context"

	authorizationv1 "k8s.io/api/authorization/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

// CheckPermission can be used to check if the current user/service-account is allowed to execute a given operation
// in the cluster.
// E.g. checkPermission(client, olmv1alpha1.GroupName, "clusterserviceversions", namespace, "camel-k", "get")
//

func CheckPermission(ctx context.Context, client kubernetes.Interface,
	group, resource, namespace, name, verb string) (bool, error) {
	sarReview := &authorizationv1.SelfSubjectAccessReview{
		Spec: authorizationv1.SelfSubjectAccessReviewSpec{
			ResourceAttributes: &authorizationv1.ResourceAttributes{
				Group:     group,
				Resource:  resource,
				Namespace: namespace,
				Name:      name,
				Verb:      verb,
			},
		},
	}

	sar, err := client.AuthorizationV1().SelfSubjectAccessReviews().Create(ctx, sarReview, metav1.CreateOptions{})
	if err != nil {
		if k8serrors.IsForbidden(err) {
			return false, nil
		}
		return false, err
	}

	return sar.Status.Allowed, nil
}
