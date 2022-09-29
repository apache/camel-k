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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/kubernetes"

	util "github.com/apache/camel-k/pkg/util/kubernetes"
)

// IsInstalled returns true if we are connected to a cluster with Knative installed.
func IsInstalled(c kubernetes.Interface) (bool, error) {
	for _, api := range getRequiredKnativeGroupVersions() {
		if installed, err := isInstalled(c, api); err != nil {
			return false, err
		} else if installed {
			return true, nil
		}
	}
	return false, nil
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

func getRequiredKnativeGroupVersions() []schema.GroupVersion {
	apis := make(map[schema.GroupVersion]bool)
	res := make([]schema.GroupVersion, 0)
	for _, gvk := range RequiredKinds {
		if !apis[gvk.GroupVersion()] {
			apis[gvk.GroupVersion()] = true
			res = append(res, gvk.GroupVersion())
		}
	}
	return res
}
