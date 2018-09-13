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
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/build"
	"github.com/apache/camel-k/pkg/build/api"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// NewBuildAction create an action that handles integration build
func NewBuildAction(ctx context.Context, namespace string) IntegrationAction {
	return &buildAction{
		buildManager: build.NewManager(ctx, namespace),
	}
}

type buildAction struct {
	buildManager *build.Manager
}

func (action *buildAction) Name() string {
	return "build"
}

func (action *buildAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuilding
}

func (action *buildAction) Handle(integration *v1alpha1.Integration) error {
	ctx, err := LookupContextForIntegration(integration)
	if err != nil {
		//TODO: we may need to add a wait strategy, i.e give up after some time
		return err
	}


	if ctx != nil {
		if ctx.Status.Phase == v1alpha1.IntegrationContextPhaseReady {
			target := integration.DeepCopy()
			target.Status.Image = ctx.Status.Image
			target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
			return sdk.Update(target)
		}

		return nil
	}

	buildIdentifier := api.BuildIdentifier{
		Name:      integration.Name,
		Qualifier: integration.Status.Digest,
	}
	buildResult := action.buildManager.Get(buildIdentifier)
	if buildResult.Status == api.BuildStatusNotRequested {
		action.buildManager.Start(api.BuildSource{
			Identifier: buildIdentifier,
			Code: api.Code{
				Name:     integration.Spec.Source.Name,
				Content:  integration.Spec.Source.Content,
				Language: integration.Spec.Source.Language,
			},
			Dependencies: integration.Spec.Dependencies,
		})
		logrus.Info("Build started")
	} else if buildResult.Status == api.BuildStatusError {
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseError
		return sdk.Update(target)
	} else if buildResult.Status == api.BuildStatusCompleted {
		target := integration.DeepCopy()
		target.Status.Image = buildResult.Image
		target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
		return sdk.Update(target)
	}

	return nil
}
