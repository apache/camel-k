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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/metadata"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
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
	for i := range target.Spec.Sources {
		// extract metadata
		s := &target.Spec.Sources[i]

		meta := metadata.Extract(*s)
		s.Language = meta.Language
	}

	// execute custom initialization
	if _, err := trait.Apply(target, nil); err != nil {
		return err
	}

	// update the status
	logrus.Info("Integration ", target.Name, " transitioning to state ", v1alpha1.IntegrationPhaseBuilding)
	target.Status.Phase = v1alpha1.IntegrationPhaseBuilding
	dgst, err := digest.ComputeForIntegration(integration)
	if err != nil {
		return err
	}
	target.Status.Digest = dgst
	return sdk.Update(target)
}
