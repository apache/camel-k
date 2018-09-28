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
	platformutils "github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/openshift"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// NewInitializeAction returns a action that initializes the platform configuration when not provided by the user
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(platform *v1alpha1.IntegrationPlatform) bool {
	return platform.Status.Phase == "" || platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseDuplicate
}

func (action *initializeAction) Handle(platform *v1alpha1.IntegrationPlatform) error {
	target := platform.DeepCopy()

	duplicate, err := action.isDuplicate(platform)
	if err != nil {
		return err
	}
	if duplicate {
		// another platform already present in the namespace
		if platform.Status.Phase != v1alpha1.IntegrationPlatformPhaseDuplicate {
			target := platform.DeepCopy()
			logrus.Info("Platform ", target.Name, " transitioning to state ", v1alpha1.IntegrationPlatformPhaseDuplicate)
			target.Status.Phase = v1alpha1.IntegrationPlatformPhaseDuplicate
			return sdk.Update(target)
		}
		return nil
	}

	// update missing fields in the resource
	if target.Spec.Cluster == "" {
		// determine the kind of cluster the platform in installed into
		if isOpenshift, err := openshift.IsOpenShift(); err != nil {
			return err
		} else if isOpenshift {
			target.Spec.Cluster = v1alpha1.IntegrationPlatformClusterOpenShift
		} else {
			target.Spec.Cluster = v1alpha1.IntegrationPlatformClusterKubernetes
		}
	}

	if target.Spec.Build.PublishStrategy == "" {
		if target.Spec.Cluster == v1alpha1.IntegrationPlatformClusterOpenShift {
			target.Spec.Build.PublishStrategy = v1alpha1.IntegrationPlatformBuildPublishStrategyS2I
		} else {
			target.Spec.Build.PublishStrategy = v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko
		}
	}

	if target.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko && target.Spec.Build.Registry == "" {
		return errors.New("no registry specified for publishing images")
	}

	// next status
	logrus.Info("Platform ", target.Name, " transitioning to state ", v1alpha1.IntegrationPlatformPhaseCreating)
	target.Status.Phase = v1alpha1.IntegrationPlatformPhaseCreating
	return sdk.Update(target)
}

func (action *initializeAction) isDuplicate(thisPlatform *v1alpha1.IntegrationPlatform) (bool, error) {
	platforms, err := platformutils.ListPlatforms(thisPlatform.Namespace)
	if err != nil {
		return false, err
	}
	for _, platform := range platforms.Items {
		if platform.Name != thisPlatform.Name && platformutils.IsActive(&platform) {
			return true, nil
		}
	}

	return false, nil
}
