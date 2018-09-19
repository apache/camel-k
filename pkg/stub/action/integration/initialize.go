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

package action

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/discover"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"sort"
)

// InitializeAction initializes the integration status to trigger the deployment
type InitializeAction struct {
}

// NewInitializeAction creates a new inititialize action
func NewInitializeAction() IntegrationAction {
	return &InitializeAction{}
}

// Name returns a common name of the action
func (b *InitializeAction) Name() string {
	return "initialize"
}

// CanHandle tells whether this action can handle the integration
func (b *InitializeAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == ""
}

// Handle handles the integratios
func (b *InitializeAction) Handle(integration *v1alpha1.Integration) error {
	target := integration.DeepCopy()
	// set default values
	if target.Spec.Replicas == nil {
		var defaultReplicas int32 = 1
		target.Spec.Replicas = &defaultReplicas
	}
	// set the correct language
	language := discover.Language(target.Spec.Source)
	target.Spec.Source.Language = language
	// discover dependencies
	if target.Spec.DependenciesAutoDiscovery == nil {
		var autoDiscoveryDependencies = true
		target.Spec.DependenciesAutoDiscovery = &autoDiscoveryDependencies
	}
	if *target.Spec.DependenciesAutoDiscovery {
		discovered := discover.Dependencies(target.Spec.Source)
		target.Spec.Dependencies = b.mergeDependencies(target.Spec.Dependencies, discovered)
	}
	// sort the dependencies to get always the same list if they don't change
	sort.Strings(target.Spec.Dependencies)
	// update the status
	target.Status.Phase = v1alpha1.IntegrationPhaseBuilding
	target.Status.Digest = digest.ComputeForIntegration(integration)
	return sdk.Update(target)
}

func (b *InitializeAction) mergeDependencies(list1 []string, list2 []string) []string {
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
