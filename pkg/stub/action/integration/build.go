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

type BuildAction struct {
	buildManager *build.BuildManager
}

func NewBuildAction(ctx context.Context, namespace string) IntegrationAction {
	return &BuildAction{
		buildManager: build.NewBuildManager(ctx, namespace),
	}
}

func (b *BuildAction) Name() string {
	return "build"
}

func (b *BuildAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuilding
}

func (b *BuildAction) Handle(integration *v1alpha1.Integration) error {
	buildIdentifier := api.BuildIdentifier{
		Name:   integration.Name,
		Digest: integration.Status.Digest,
	}
	buildResult := b.buildManager.Get(buildIdentifier)
	if buildResult.Status == api.BuildStatusNotRequested {
		b.buildManager.Start(api.BuildSource{
			Identifier: buildIdentifier,
			Code: api.Code{
				Name:    *integration.Spec.Source.Name,
				Content: *integration.Spec.Source.Content,
			}, // FIXME possible panic
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
