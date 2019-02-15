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

package integrationcontext

import (
	"context"
	"errors"
	"fmt"

	"github.com/apache/camel-k/pkg/util/cancellable"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/util/kubernetes"

	"github.com/apache/camel-k/pkg/trait"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewBuildAction creates a new build request handling action for the context
func NewBuildAction() Action {
	return &buildAction{}
}

type buildAction struct {
	baseAction
}

func (action *buildAction) Name() string {
	return "build-submitted"
}

func (action *buildAction) CanHandle(ictx *v1alpha1.IntegrationContext) bool {
	if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildSubmitted {
		return true
	}
	if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildRunning {
		return true
	}

	return false
}

func (action *buildAction) Handle(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildSubmitted {
		return action.handleBuildSubmitted(ctx, ictx)
	}
	if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildRunning {
		return action.handleBuildRunning(ctx, ictx)
	}

	return nil
}

func (action *buildAction) handleBuildRunning(_ context.Context, ictx *v1alpha1.IntegrationContext) error {
	b, err := platform.GetPlatformBuilder(action.client, ictx.Namespace)
	if err != nil {
		return err
	}

	if b.IsBuilding(ictx.ObjectMeta) {
		action.L.Info("Build running")
	}

	return nil
}

func (action *buildAction) handleBuildSubmitted(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	b, err := platform.GetPlatformBuilder(action.client, ictx.Namespace)
	if err != nil {
		return err
	}

	if !b.IsBuilding(ictx.ObjectMeta) {
		p, err := platform.GetCurrentPlatform(ctx, action.client, ictx.Namespace)
		if err != nil {
			return err
		}
		env, err := trait.Apply(ctx, action.client, nil, ictx)
		if err != nil {
			return err
		}

		// assume there's no duplication nor conflict for now
		repositories := make([]string, 0, len(ictx.Spec.Repositories)+len(p.Spec.Build.Repositories))
		repositories = append(repositories, ictx.Spec.Repositories...)
		repositories = append(repositories, p.Spec.Build.Repositories...)

		if env.CamelCatalog == nil {
			return errors.New("undefined camel catalog")
		}

		// the context given to the handler is per reconcile loop and as the build
		// happens asynchronously, a new context has to be created. the new context
		// can be used also to stop the build.
		r := builder.Request{
			C:            cancellable.NewContext(),
			Catalog:      env.CamelCatalog,
			Meta:         ictx.ObjectMeta,
			Dependencies: ictx.Spec.Dependencies,
			Repositories: repositories,
			Steps:        env.Steps,
			BuildDir:     env.BuildDir,
			Platform:     env.Platform.Spec,
		}

		b.Submit(r, func(result *builder.Result) {
			//
			// this function is invoked synchronously for every state change to avoid
			// leaving one context not fully updated when the incremental builder search
			// for a compatible/base image
			//
			if err := action.handleBuildStateChange(result.Request.C, result); err != nil {
				action.L.Error(err, "Error while building integration context")
			}
		})
	}

	return nil
}

func (action *buildAction) handleBuildStateChange(ctx context.Context, res *builder.Result) error {
	//
	// Get the latest status of the context
	//
	target, err := kubernetes.GetIntegrationContext(ctx, action.client, res.Request.Meta.Name, res.Request.Meta.Namespace)
	if err != nil || target == nil {
		return err
	}

	switch res.Status {
	case builder.StatusSubmitted:
		action.L.Info("Build submitted")
	case builder.StatusStarted:
		target.Status.Phase = v1alpha1.IntegrationContextPhaseBuildRunning

		action.L.Info("IntegrationContext state transition", "phase", target.Status.Phase)

		return action.client.Update(ctx, target)
	case builder.StatusError:
		// we should ensure that the integration context is still in the right
		// phase, if not there is a chance that the context has been modified
		// by the user
		if target.Status.Phase != v1alpha1.IntegrationContextPhaseBuildRunning {

			// terminate the build
			res.Request.C.Cancel()

			return fmt.Errorf("found context %s not the an expected phase (expectd=%s, found=%s)",
				res.Request.Meta.Name,
				string(v1alpha1.IntegrationContextPhaseBuildRunning),
				string(target.Status.Phase),
			)
		}

		target.Status.Phase = v1alpha1.IntegrationContextPhaseBuildFailureRecovery

		if target.Status.Failure == nil {
			target.Status.Failure = &v1alpha1.Failure{
				Reason: res.Error.Error(),
				Time:   metav1.Now(),
				Recovery: v1alpha1.FailureRecovery{
					Attempt:    0,
					AttemptMax: 5,
				},
			}
		}

		action.L.Error(res.Error, "IntegrationContext state transition", "phase", target.Status.Phase)

		return action.client.Update(ctx, target)
	case builder.StatusCompleted:
		// we should ensure that the integration context is still in the right
		// phase, if not there is a chance that the context has been modified
		// by the user
		if target.Status.Phase != v1alpha1.IntegrationContextPhaseBuildRunning {
			// terminate the build
			res.Request.C.Cancel()

			return fmt.Errorf("found context %s not in the expected phase (expectd=%s, found=%s)",
				res.Request.Meta.Name,
				string(v1alpha1.IntegrationContextPhaseBuildRunning),
				string(target.Status.Phase),
			)
		}

		target.Status.BaseImage = res.BaseImage
		target.Status.Image = res.Image
		target.Status.PublicImage = res.PublicImage
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

		action.L.Info("IntegrationContext state transition", "phase", target.Status.Phase)
		if err := action.client.Update(ctx, target); err != nil {
			return err
		}

		action.L.Info("Inform integrations about context state change")
		if err := action.informIntegrations(ctx, target); err != nil {
			return err
		}
	}

	return nil
}

// informIntegrations triggers the processing of all integrations waiting for this context to be built
func (action *buildAction) informIntegrations(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	list := v1alpha1.NewIntegrationList()
	err := action.client.List(ctx, &k8sclient.ListOptions{Namespace: ictx.Namespace}, &list)
	if err != nil {
		return err
	}
	for _, integration := range list.Items {
		integration := integration // pin
		if integration.Status.Context != ictx.Name {
			continue
		}

		if integration.Annotations == nil {
			integration.Annotations = make(map[string]string)
		}
		integration.Annotations["camel.apache.org/context.digest"] = ictx.Status.Digest
		err = action.client.Update(ctx, &integration)
		if err != nil {
			return err
		}
	}
	return nil
}
