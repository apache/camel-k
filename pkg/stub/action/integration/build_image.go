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
	"fmt"
	"path"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/util/digest"

	"github.com/apache/camel-k/pkg/trait"

	"github.com/apache/camel-k/pkg/builder"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/pkg/platform"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewBuildImageAction create an action that handles integration image build
func NewBuildImageAction(ctx context.Context, namespace string) Action {
	return &buildImageAction{
		Context:   ctx,
		namespace: namespace,
	}
}

type buildImageAction struct {
	context.Context
	namespace string
}

func (action *buildImageAction) Name() string {
	return "build-image"
}

func (action *buildImageAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuildingImage
}

func (action *buildImageAction) Handle(integration *v1alpha1.Integration) error {

	// in this phase the integration need to be associated to a context whose image
	// will be used as base image for the integration images
	if integration.Spec.Context == "" {
		return fmt.Errorf("context is not set for integration: %s", integration.Name)
	}

	// look-up the integration context associated to this integration, this is needed
	// to determine the base image
	ctx := v1alpha1.NewIntegrationContext(integration.Namespace, integration.Spec.Context)
	if err := sdk.Get(&ctx); err != nil {
		return errors.Wrapf(err, "unable to find integration context %s, %s", ctx.Name, err)
	}

	b, err := platform.GetPlatformBuilder(action.Context, action.namespace)
	if err != nil {
		return err
	}
	env, err := trait.Apply(integration, &ctx)
	if err != nil {
		return err
	}

	// This build do not require to determine dependencies nor a project, the builder
	// step do remove them
	r := builder.Request{
		Meta:     integration.ObjectMeta,
		Steps:    env.Steps,
		Platform: env.Platform.Spec,
		Image:    ctx.Status.Image,
	}

	// Sources are added as part of the standard deployment bits
	r.Resources = make([]builder.Resource, 0, len(integration.Spec.Sources))

	for _, source := range integration.Spec.Sources {
		r.Resources = append(r.Resources, builder.Resource{
			Content: []byte(source.Content),
			Target:  path.Join("sources", source.Name),
		})
	}

	res := b.Submit(r)

	switch res.Status {
	case builder.StatusSubmitted:
		logrus.Info("Build submitted")
	case builder.StatusStarted:
		logrus.Info("Build started")
	case builder.StatusError:
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseError

		logrus.Info("Integration ", target.Name, " transitioning to state ", target.Status.Phase)

		// remove the build from cache
		defer b.Purge(r)

		return sdk.Update(target)
	case builder.StatusCompleted:
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
		target.Status.Image = res.Image

		dgst, err := digest.ComputeForIntegration(integration)
		if err != nil {
			return err
		}

		target.Status.Digest = dgst

		logrus.Info("Integration ", target.Name, " transitioning to state ", target.Status.Phase)

		// remove the build from cache
		defer b.Purge(r)

		if err := sdk.Update(target); err != nil {
			return err
		}
	}

	return nil
}
