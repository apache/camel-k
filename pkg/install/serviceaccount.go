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
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// ServiceAccountRoles installs the service account and related roles in the given namespace
func ServiceAccountRoles(ctx context.Context, c client.Client, namespace string) error {
	isOpenshift, err := openshift.IsOpenShift(c)
	if err != nil {
		return err
	}
	if isOpenshift {
		if err := installServiceAccountRolesOpenshift(ctx, c, namespace); err != nil {
			return err
		}
	} else {
		if err := installServiceAccountRolesKubernetes(ctx, c, namespace); err != nil {
			return err
		}
	}
	// Install Knative resources if required
	isKnative, err := knative.IsInstalled(ctx, c)
	if err != nil {
		return err
	}
	if isKnative {
		return installKnative(ctx, c, namespace, nil)
	}
	return nil
}

func installServiceAccountRolesOpenshift(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, IdentityResourceCustomizer,
		"operator-service-account.yaml",
		"operator-role-openshift.yaml",
		"operator-role-binding.yaml",
	)
}

func installServiceAccountRolesKubernetes(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, IdentityResourceCustomizer,
		"operator-service-account.yaml",
		"operator-role-kubernetes.yaml",
		"operator-role-binding.yaml",
	)
}
