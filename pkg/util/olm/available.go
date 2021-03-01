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

package olm

import (
	"context"

	kubernetesutils "github.com/apache/camel-k/pkg/util/kubernetes"
	v1 "github.com/operator-framework/api/pkg/operators/v1"
	v1alpha1 "github.com/operator-framework/api/pkg/operators/v1alpha1"
	v1alpha2 "github.com/operator-framework/api/pkg/operators/v1alpha2"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"
)

// IsAPIAvailable returns true if we are connected to a cluster with OLM installed
//
// This method should not be called from the operator, as it might require permissions that are not available.
func IsAPIAvailable(ctx context.Context, c kubernetes.Interface, namespace string) (bool, error) {
	// check some Knative APIs
	for _, api := range getOLMGroupVersions() {
		if installed, err := isAvailable(c, api); err != nil {
			return false, err
		} else if installed {
			return true, nil
		}
	}

	return false, nil
}

func isAvailable(c kubernetes.Interface, api schema.GroupVersion) (bool, error) {
	_, err := c.Discovery().ServerResourcesForGroupVersion(api.String())
	if err != nil && (k8serrors.IsNotFound(err) || kubernetesutils.IsUnknownAPIError(err)) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func getOLMGroupVersions() []schema.GroupVersion {
	return []schema.GroupVersion{
		v1alpha1.SchemeGroupVersion,
		v1alpha2.SchemeGroupVersion,
		v1.SchemeGroupVersion,
	}
}
