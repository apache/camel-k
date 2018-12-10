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

package context

import (
	"context"

	"github.com/apache/camel-k/pkg/trait"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// NewBuildAction creates a new build handling action for the context
func NewBuildAction(ctx context.Context) Action {
	return &buildAction{
		ctx,
	}
}

type buildAction struct {
	context.Context
}

func (action *buildAction) Name() string {
	return "build"
}

func (action *buildAction) CanHandle(context *v1alpha1.IntegrationContext) bool {
	return context.Status.Phase == v1alpha1.IntegrationContextPhaseBuilding
}

func (action *buildAction) Handle(context *v1alpha1.IntegrationContext) error {
	b, err := platform.GetPlatformBuilder(action.Context, context.Namespace)
	if err != nil {
		return err
	}
	env, err := trait.Apply(nil, context)
	if err != nil {
		return err
	}

	r := builder.Request{
		Meta:         context.ObjectMeta,
		Dependencies: context.Spec.Dependencies,
		Repositories: context.Spec.Repositories,
		Steps:        env.Steps,
		BuildDir:     env.BuildDir,
		Platform:     env.Platform.Spec,
	}

	res := b.Submit(r)
	switch res.Status {
	case builder.StatusSubmitted:
		logrus.Info("Build submitted")
	case builder.StatusStarted:
		logrus.Info("Build started")
	case builder.StatusError:
		target := context.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationContextPhaseError

		logrus.Infof("Context %s transitioning to state %s, reason: %s", target.Name, target.Status.Phase, res.Error.Error())

		// remove the build from cache
		defer b.Purge(r)

		return sdk.Update(target)
	case builder.StatusCompleted:
		target := context.DeepCopy()
		target.Status.Image = res.Image
		target.Status.Phase = v1alpha1.IntegrationContextPhaseReady
		target.Status.Artifacts = make([]v1alpha1.Artifact, 0, len(res.Artifacts))

		for _, a := range res.Artifacts {
			// do not include artifact location
			target.Status.Artifacts = append(target.Status.Artifacts, v1alpha1.Artifact{
				ID:       a.ID,
				Location: "",
				Target:   a.Target,
			})
		}

		logrus.Info("Context ", target.Name, " transitioning to state ", target.Status.Phase)

		// remove the build from cache
		defer b.Purge(r)

		if err := sdk.Update(target); err != nil {
			return err
		}
		if err := action.informIntegrations(target); err != nil {
			return err
		}
	}

	return nil
}

// informIntegrations triggers the processing of all integrations waiting for this context to be built
func (action *buildAction) informIntegrations(context *v1alpha1.IntegrationContext) error {
	list := v1alpha1.NewIntegrationList()
	err := sdk.List(context.Namespace, &list, sdk.WithListOptions(&metav1.ListOptions{}))
	if err != nil {
		return err
	}
	for _, integration := range list.Items {
		integration := integration // pin
		if integration.Spec.Context != context.Name {
			continue
		}

		if integration.Annotations == nil {
			integration.Annotations = make(map[string]string)
		}
		integration.Annotations["camel.apache.org/context.digest"] = context.Status.Digest
		err = sdk.Update(&integration)
		if err != nil {
			return err
		}
	}
	return nil
}
