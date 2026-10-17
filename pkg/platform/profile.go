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

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

// ApplyIntegrationProfile resolves integration profile from given object.
func ApplyIntegrationProfile(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationProfile, error) {
	profile, err := findIntegrationProfile(ctx, c, o)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	return profile, nil
}

// findIntegrationProfile finds profile from given resource annotations
// and resolves the profile in given resource namespace, falling back to
// namespace annotation, "default" profile, or operator defaults.
func findIntegrationProfile(ctx context.Context, c k8sclient.Reader, o k8sclient.Object) (*v1.IntegrationProfile, error) {
	namespace := o.GetNamespace()

	// 1. User provided profile (via annotation on the resource)
	if profileName := v1.GetIntegrationProfileAnnotation(o); profileName != "" {
		return kubernetes.GetIntegrationProfile(ctx, c, profileName, namespace)
	}

	// 2. Namespace annotated profile (via namespace annotation)
	if namespace != "" {
		var ns corev1.Namespace
		err := c.Get(ctx, k8sclient.ObjectKey{Name: namespace}, &ns)
		if err == nil {
			if profileName := v1.GetIntegrationProfileAnnotation(&ns); profileName != "" {
				return kubernetes.GetIntegrationProfile(ctx, c, profileName, namespace)
			}
		} else if !k8serrors.IsNotFound(err) && !k8serrors.IsForbidden(err) && !k8serrors.IsUnauthorized(err) {
			return nil, err
		}
	}

	// 3. "default" profile in the same namespace
	if namespace != "" {
		profile, err := kubernetes.GetIntegrationProfile(ctx, c, v1.DefaultIntegrationProfileName, namespace)
		if err == nil {
			return profile, nil
		}
		if !k8serrors.IsNotFound(err) {
			return nil, err
		}
	}

	// 4. Default operator configuration (no profile found)
	return nil, nil
}
