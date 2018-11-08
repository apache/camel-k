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
	"errors"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// GetCurrentPlatform returns the currently installed platform
func GetCurrentPlatform(namespace string) (*v1alpha1.IntegrationPlatform, error) {
	lst, err := ListPlatforms(namespace)
	if err != nil {
		return nil, err
	}

	for _, platform := range lst.Items {
		if IsActive(&platform) {
			return &platform, nil
		}
	}
	return nil, errors.New("no active integration platforms found in the namespace")
}

// ListPlatforms returns all platforms installed in a given namespace (only one will be active)
func ListPlatforms(namespace string) (*v1alpha1.IntegrationPlatformList, error) {
	lst := v1alpha1.NewIntegrationPlatformList()
	if err := sdk.List(namespace, &lst); err != nil {
		return nil, err
	}
	return &lst, nil
}

// IsActive determines if the given platform is being used
func IsActive(p *v1alpha1.IntegrationPlatform) bool {
	return p.Status.Phase != "" && p.Status.Phase != v1alpha1.IntegrationPlatformPhaseDuplicate
}

// GetProfile returns the current profile of the platform (if present) or computes it
func GetProfile(p *v1alpha1.IntegrationPlatform) v1alpha1.TraitProfile {
	if p.Spec.Profile != "" {
		return p.Spec.Profile
	}
	switch p.Spec.Cluster {
	case v1alpha1.IntegrationPlatformClusterKubernetes:
		return v1alpha1.TraitProfileKubernetes
	case v1alpha1.IntegrationPlatformClusterOpenShift:
		return v1alpha1.TraitProfileOpenShift
	}
	return ""
}