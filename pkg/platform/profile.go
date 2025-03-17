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

package platform

import (
	"context"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// ApplyIntegrationProfile resolves integration profile from given object.
func ApplyIntegrationProfile(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationProfile, error) {
	profile, err := findIntegrationProfile(ctx, c, o)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	return profile, nil
}

// findIntegrationProfile finds profile from given resource annotations and resolves the profile in given resource namespace or operator namespace as a fallback option.
func findIntegrationProfile(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationProfile, error) {
	if profileName := v1.GetIntegrationProfileAnnotation(o); profileName != "" {
		namespace := v1.GetIntegrationProfileNamespaceAnnotation(o)
		if namespace == "" {
			namespace = o.GetNamespace()
		}

		profile, err := kubernetes.GetIntegrationProfile(ctx, c, profileName, namespace)
		if err != nil && k8serrors.IsNotFound(err) {
			operatorNamespace := GetOperatorNamespace()
			if operatorNamespace != "" && operatorNamespace != namespace {
				profile, err = kubernetes.GetIntegrationProfile(ctx, c, profileName, operatorNamespace)
			}
		}
		return profile, err
	}

	return nil, nil
}
