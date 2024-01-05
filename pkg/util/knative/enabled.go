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

package knative

import (
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	util "github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

// IsRefKindInstalled returns true if the cluster has the referenced Kind installed.
func IsRefKindInstalled(c kubernetes.Interface, ref corev1.ObjectReference) (bool, error) {
	if installed, err := isInstalled(c, ref.GroupVersionKind().GroupVersion()); err != nil {
		return false, err
	} else if installed {
		return true, nil
	}
	return false, nil
}

// IsServingInstalled returns true if we are connected to a cluster with Knative Serving installed.
func IsServingInstalled(c kubernetes.Interface) (bool, error) {
	return IsRefKindInstalled(c, corev1.ObjectReference{
		Kind:       "Service",
		APIVersion: "serving.knative.dev/v1",
	})
}

// IsEventingInstalled returns true if we are connected to a cluster with Knative Eventing installed.
func IsEventingInstalled(c kubernetes.Interface) (bool, error) {
	return IsRefKindInstalled(c, corev1.ObjectReference{
		Kind:       "Broker",
		APIVersion: "eventing.knative.dev/v1",
	})
}

func isInstalled(c kubernetes.Interface, api schema.GroupVersion) (bool, error) {
	_, err := c.Discovery().ServerResourcesForGroupVersion(api.String())
	if err != nil && (k8serrors.IsNotFound(err) || util.IsUnknownAPIError(err)) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}
