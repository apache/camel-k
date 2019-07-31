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
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
)

// NewDeployAction create an action that handles integration deploy
func NewDeployAction() Action {
	return &deployAction{}
}

type deployAction struct {
	baseAction
}

func (action *deployAction) Name() string {
	return "deploy"
}

func (action *deployAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (action *deployAction) Handle(ctx context.Context, integration *v1alpha1.Integration) (*v1alpha1.Integration, error) {
	if integration.Status.Kit == "" {
		return nil, errors.Errorf("no kit set on integration %s", integration.Name)
	}

	kit, err := kubernetes.GetIntegrationKit(ctx, action.client, integration.Status.Kit, integration.Namespace)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to find integration kit %s, %s", integration.Status.Kit, err)
	}

	if _, err := trait.Apply(ctx, action.client, integration, kit); err != nil {
		return err
	}

	target := integration.DeepCopy()
	target.Status.Phase = v1alpha1.IntegrationPhaseRunning

	return integration, nil
}
