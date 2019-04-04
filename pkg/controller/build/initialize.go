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

package build

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewInitializeAction creates a new initialize action
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *initializeAction) Name() string {
	return "initialize"
}

// CanHandle tells whether this action can handle the build
func (action *initializeAction) CanHandle(build *v1alpha1.Build) bool {
	return build.Status.Phase == v1alpha1.BuildPhaseInitial
}

// Handle handles the builds
func (action *initializeAction) Handle(ctx context.Context, build *v1alpha1.Build) error {
	target := build.DeepCopy()
	target.Status.Phase = v1alpha1.BuildPhaseScheduling
	action.L.Info("Build state transition", "phase", target.Status.Phase)

	return action.client.Status().Update(ctx, target)
}
