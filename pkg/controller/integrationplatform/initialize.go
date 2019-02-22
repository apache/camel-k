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

package integrationplatform

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/defaults"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// NewInitializeAction returns a action that initializes the platform configuration when not provided by the user
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(platform *v1alpha1.IntegrationPlatform) bool {
	return platform.Status.Phase == "" || platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseDuplicate
}

func (action *initializeAction) Handle(ctx context.Context, ip *v1alpha1.IntegrationPlatform) error {
	target := ip.DeepCopy()

	duplicate, err := action.isDuplicate(ctx, ip)
	if err != nil {
		return err
	}
	if duplicate {
		// another platform already present in the namespace
		if ip.Status.Phase != v1alpha1.IntegrationPlatformPhaseDuplicate {
			target := ip.DeepCopy()
			target.Status.Phase = v1alpha1.IntegrationPlatformPhaseDuplicate

			action.L.Info("IntegrationPlatform state transition", "phase", target.Status.Phase)

			return action.client.Status().Update(ctx, target)
		}
		return nil
	}

	// update missing fields in the resource
	if target.Spec.Cluster == "" {
		// determine the kind of cluster the platform is installed into
		isOpenShift, err := openshift.IsOpenShift(action.client)
		switch {
		case err != nil:
			return err
		case isOpenShift:
			target.Spec.Cluster = v1alpha1.IntegrationPlatformClusterOpenShift
		default:
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
		action.L.Info("No registry specified for publishing images")
	}

	if target.Spec.Profile == "" {
		target.Spec.Profile = platform.GetProfile(target)
	}
	if target.Spec.Build.CamelVersion == "" {
		target.Spec.Build.CamelVersion = defaults.CamelVersionConstraint
	}
	if target.Spec.Build.RuntimeVersion == "" {
		target.Spec.Build.RuntimeVersion = defaults.RuntimeVersion
	}
	if target.Spec.Build.BaseImage == "" {
		target.Spec.Build.BaseImage = defaults.BaseImage
	}
	if target.Spec.Build.LocalRepository == "" {
		target.Spec.Build.LocalRepository = defaults.LocalRepository
	}

	action.L.Infof("CamelVersion set to %s", target.Spec.Build.CamelVersion)
	action.L.Infof("RuntimeVersion set to %s", target.Spec.Build.RuntimeVersion)
	action.L.Infof("BaseImage set to %s", target.Spec.Build.BaseImage)
	action.L.Infof("LocalRepository set to %s", target.Spec.Build.LocalRepository)

	for i, r := range target.Spec.Build.Repositories {
		if i == 0 {
			action.L.Info("Repositories:")
		}

		action.L.Infof("    %d - %s", i, r)
	}

	err = action.client.Update(ctx, target)
	if err != nil {
		return err
	}

	if target.Spec.Build.PublishStrategy == v1alpha1.IntegrationPlatformBuildPublishStrategyKaniko {
		// create the Kaniko warmer pod that caches the base image into the Camel K builder volume
		action.L.Info("Create Kaniko cache warmer pod")
		err := createKanikoCacheWarmerPod(ctx, action.client, target)
		if err != nil {
			return err
		}
		target.Status.Phase = v1alpha1.IntegrationPlatformPhaseWarming
	} else {
		target.Status.Phase = v1alpha1.IntegrationPlatformPhaseCreating
	}

	// next phase
	action.L.Info("IntegrationPlatform state transition", "phase", target.Status.Phase)
	return action.client.Status().Update(ctx, target)
}

func (action *initializeAction) isDuplicate(ctx context.Context, thisPlatform *v1alpha1.IntegrationPlatform) (bool, error) {
	platforms, err := platform.ListPlatforms(ctx, action.client, thisPlatform.Namespace)
	if err != nil {
		return false, err
	}
	for _, p := range platforms.Items {
		p := p // pin
		if p.Name != thisPlatform.Name && platform.IsActive(&p) {
			return true, nil
		}
	}

	return false, nil
}
