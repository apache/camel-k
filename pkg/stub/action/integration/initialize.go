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

package integration

import (
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/sirupsen/logrus"
	"sort"

	"github.com/apache/camel-k/pkg/util"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
)

// NewInitializeAction creates a new inititialize action
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
}

// Name returns a common name of the action
func (action *initializeAction) Name() string {
	return "initialize"
}

// CanHandle tells whether this action can handle the integration
func (action *initializeAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == ""
}

// Handle handles the integrations
func (action *initializeAction) Handle(integration *v1alpha1.Integration) error {
	// The integration platform needs to be ready before starting to create integrations
	if pl, err := platform.GetCurrentPlatform(integration.Namespace); err != nil || pl.Status.Phase != v1alpha1.IntegrationPlatformPhaseReady {
		logrus.Info("Waiting for a integration platform to be ready")
		return nil
	}

	target := integration.DeepCopy()
	// set default values
	if target.Spec.Replicas == nil {
		var defaultReplicas int32 = 1
		target.Spec.Replicas = &defaultReplicas
	}
	// extract metadata
	meta := metadata.Extract(target.Spec.Source)

	// set the correct language
	target.Spec.Source.Language = meta.Language

	if !util.StringSliceExists(target.Spec.Dependencies, "camel:core") {
		target.Spec.Dependencies = append(target.Spec.Dependencies, "camel:core")
	}

	// discover dependencies
	if target.Spec.DependenciesAutoDiscovery == nil {
		var autoDiscoveryDependencies = true
		target.Spec.DependenciesAutoDiscovery = &autoDiscoveryDependencies
	}
	if *target.Spec.DependenciesAutoDiscovery {
		target.Spec.Dependencies = action.mergeDependencies(target.Spec.Dependencies, meta.Dependencies)
	}
	// sort the dependencies to get always the same list if they don't change
	sort.Strings(target.Spec.Dependencies)
	// update the status
	logrus.Info("Integration ", target.Name, " transitioning to state ", v1alpha1.IntegrationPhaseBuilding)
	target.Status.Phase = v1alpha1.IntegrationPhaseBuilding
	target.Status.Digest = digest.ComputeForIntegration(integration)
	return sdk.Update(target)
}

func (action *initializeAction) mergeDependencies(list1 []string, list2 []string) []string {
	set := make(map[string]bool, 0)
	for _, d := range list1 {
		set[d] = true
	}
	for _, d := range list2 {
		set[d] = true
	}
	ret := make([]string, 0, len(set))
	for d := range set {
		ret = append(ret, d)
	}
	return ret
}
