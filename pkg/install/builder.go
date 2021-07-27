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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
)

// BuilderServiceAccountRoles installs the builder service account and related roles in the given namespace
func BuilderServiceAccountRoles(ctx context.Context, c client.Client, namespace string, cluster v1.IntegrationPlatformCluster) error {
	if cluster == v1.IntegrationPlatformClusterOpenShift {
		if err := installBuilderServiceAccountRolesOpenShift(ctx, c, namespace); err != nil {
			return err
		}
	} else {
		if err := installBuilderServiceAccountRolesKubernetes(ctx, c, namespace); err != nil {
			return err
		}
	}
	return nil
}

func installBuilderServiceAccountRolesOpenShift(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, true, IdentityResourceCustomizer,
		"/builder/builder-service-account.yaml",
		"/builder/builder-role.yaml",
		"/builder/builder-role-binding.yaml",
		"/builder/builder-role-openshift.yaml",
		"/builder/builder-role-binding-openshift.yaml",
	)
}

func installBuilderServiceAccountRolesKubernetes(ctx context.Context, c client.Client, namespace string) error {
	return ResourcesOrCollect(ctx, c, namespace, nil, true, IdentityResourceCustomizer,
		"/builder/builder-service-account.yaml",
		"/builder/builder-role.yaml",
		"/builder/builder-role-binding.yaml",
	)
}
