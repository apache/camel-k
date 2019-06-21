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
	"github.com/apache/camel-k/pkg/util/openshift"
)

// BuilderServiceAccountRoles installs the builder service account and related roles in the given namespace
func BuilderServiceAccountRoles(ctx context.Context, c client.Client, namespace string) error {
	isOpenshift, err := openshift.IsOpenShift(c)
	if err != nil {
		return err
	}
	if isOpenshift {
		if err := installBuilderServiceAccountRolesOpenshift(ctx, c, namespace); err != nil {
			return err
		}
	} else {
		if err := installBuilderServiceAccountRolesKubernetes(ctx, c, namespace); err != nil {
			return err
		}
	}
	return nil
}

func installBuilderServiceAccountRolesOpenshift(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, IdentityResourceCustomizer,
		"builder-service-account.yaml",
		"builder-role-openshift.yaml",
		"builder-role-binding.yaml",
	)
}

func installBuilderServiceAccountRolesKubernetes(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, IdentityResourceCustomizer,
		"builder-service-account.yaml",
		"builder-role-kubernetes.yaml",
		"builder-role-binding.yaml",
	)
}
