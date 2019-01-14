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
	"github.com/apache/camel-k/pkg/platform"
	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewBuildImageAction create an action that handles integration image build
func NewBuildImageAction(namespace string) Action {
	return &buildImageAction{
		namespace: namespace,
	}
}

type buildImageAction struct {
	baseAction
	context.Context
	namespace string
}

func (action *buildImageAction) Name() string {
	return "build-image"
}

func (action *buildImageAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuildingImage
}

func (action *buildImageAction) Handle(ctx context.Context, integration *v1alpha1.Integration) error {

	// in this phase the integration need to be associated to a context whose image
	// will be used as base image for the integration images
	if integration.Status.Context == "" {
		return fmt.Errorf("context is not set for integration: %s", integration.Name)
	}

	// look-up the integration context associated to this integration, this is needed
	// to determine the base image
	ictx := v1alpha1.NewIntegrationContext(integration.Namespace, integration.Status.Context)
	ikey := k8sclient.ObjectKey{
		Namespace: integration.Namespace,
		Name:      integration.Status.Context,
	}
	if err := action.client.Get(ctx, ikey, &ictx); err != nil {
		return errors.Wrapf(err, "unable to find integration context %s, %s", ikey.Name, err)
	}

	b, err := platform.GetPlatformBuilder(action.Context, action.client, action.namespace)
	if err != nil {
		return err
	}
	env, err := trait.Apply(ctx, action.client, integration, &ictx)
	if err != nil {
		return err
	}

	// This build do not require to determine dependencies nor a project, the builder
	// step do remove them
	r := builder.Request{
		Meta:     integration.ObjectMeta,
		Steps:    env.Steps,
		BuildDir: env.BuildDir,
		Platform: env.Platform.Spec,
		Image:    ictx.Status.Image,
	}

	// Sources are added as part of the standard deployment bits
	r.Resources = make([]builder.Resource, 0, len(integration.Spec.Sources))

	for _, source := range integration.Spec.Sources {
		r.Resources = append(r.Resources, builder.Resource{
			Content: []byte(source.Content),
			Target:  path.Join("sources", source.Name),
		})
	}
	for _, resource := range integration.Spec.Resources {
		if resource.Type != v1alpha1.ResourceTypeData {
			continue
		}
		r.Resources = append(r.Resources, builder.Resource{
			Content: []byte(resource.Content),
			Target:  path.Join("resources", resource.Name),
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

		logrus.Infof("Integration %s transitioning to state %s, reason: %s", target.Name, target.Status.Phase, res.Error.Error())

		// remove the build from cache
		defer b.Purge(r)

		return action.client.Update(ctx, target)
	case builder.StatusCompleted:
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
		if res.PublicImage != "" {
			target.Status.Image = res.PublicImage
		} else {
			target.Status.Image = res.Image
		}

		dgst, err := digest.ComputeForIntegration(integration)
		if err != nil {
			return err
		}

		target.Status.Digest = dgst

		logrus.Info("Integration ", target.Name, " transitioning to state ", target.Status.Phase)

		// remove the build from cache
		defer b.Purge(r)

		if err := action.client.Update(ctx, target); err != nil {
			return err
		}
	}

	return nil
}
