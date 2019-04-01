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
	"fmt"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
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
	return ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildSubmitted ||
		ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildRunning
}

func (action *buildAction) Handle(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildSubmitted {
		return action.handleBuildSubmitted(ctx, ictx)
	} else if ictx.Status.Phase == v1alpha1.IntegrationContextPhaseBuildRunning {
		return action.handleBuildRunning(ctx, ictx)
	}

	return nil
}

func (action *buildAction) handleBuildSubmitted(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	build := &v1alpha1.Build{}
	err := action.client.Get(ctx, types.NamespacedName{Namespace: ictx.Namespace, Name: ictx.Name}, build)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if err != nil && k8serrors.IsNotFound(err) {
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

		steps := make([]string, len(env.Steps))
		for i, s := range env.Steps {
			steps[i] = s.ID()
		}

		build = &v1alpha1.Build{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "camel.apache.org/v1alpha1",
				Kind:       "Build",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: ictx.Namespace,
				Name:      ictx.Name,
			},
			Spec: v1alpha1.BuildSpec{
				Meta:           ictx.ObjectMeta,
				CamelVersion:   env.CamelCatalog.Version,
				RuntimeVersion: env.RuntimeVersion,
				//Image:          "",
				Platform:     env.Platform.Spec,
				Dependencies: ictx.Spec.Dependencies,
				Repositories: repositories,
				Steps:        steps,
				//	BuildDir:       env.BuildDir,
				//Resources:    request.Resources,
			},
		}

		// Set the integration context instance as the owner and controller
		if err := controllerutil.SetControllerReference(ictx, build, action.client.GetScheme()); err != nil {
			return err
		}

		err = action.client.Delete(ctx, build)
		if err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrap(err, "cannot delete build pod")
		}

		err = action.client.Create(ctx, build)
		if err != nil {
			return err
		}
	}

	if build.Status.Phase == v1alpha1.BuildPhaseRunning {
		target := ictx.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationContextPhaseBuildRunning

		action.L.Info("IntegrationContext state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
	}

	return nil
}

func (action *buildAction) handleBuildRunning(ctx context.Context, ictx *v1alpha1.IntegrationContext) error {
	build := &v1alpha1.Build{}
	err := action.client.Get(ctx, types.NamespacedName{Namespace: ictx.Namespace, Name: ictx.Name}, build)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	switch build.Status.Phase {

	case v1alpha1.BuildPhaseRunning:
		action.L.Info("Build running")

	case v1alpha1.BuildPhaseSucceeded:
		target := ictx.DeepCopy()
		// we should ensure that the integration context is still in the right
		// phase, if not there is a chance that the context has been modified
		// by the user
		if target.Status.Phase != v1alpha1.IntegrationContextPhaseBuildRunning {
			return fmt.Errorf("found context %s not in the expected phase (expectd=%s, found=%s)",
				build.Spec.Meta.Name,
				string(v1alpha1.IntegrationContextPhaseBuildRunning),
				string(target.Status.Phase),
			)
		}

		target.Status.BaseImage = build.Status.BaseImage
		target.Status.Image = build.Status.Image
		target.Status.PublicImage = build.Status.PublicImage
		target.Status.Phase = v1alpha1.IntegrationContextPhaseReady
		target.Status.Artifacts = make([]v1alpha1.Artifact, 0, len(build.Status.Artifacts))

		for _, a := range build.Status.Artifacts {
			// do not include artifact location
			target.Status.Artifacts = append(target.Status.Artifacts, v1alpha1.Artifact{
				ID:       a.ID,
				Location: "",
				Target:   a.Target,
			})
		}

		action.L.Info("IntegrationContext state transition", "phase", target.Status.Phase)
		if err := action.client.Status().Update(ctx, target); err != nil {
			return err
		}

		action.L.Info("Inform integrations about context state change")
		if err := action.informIntegrations(ctx, target); err != nil {
			return err
		}

	case v1alpha1.BuildPhaseFailed, v1alpha1.BuildPhaseInterrupted:
		target := ictx.DeepCopy()
		// we should ensure that the integration context is still in the right
		// phase, if not there is a chance that the context has been modified
		// by the user
		if target.Status.Phase != v1alpha1.IntegrationContextPhaseBuildRunning {
			return fmt.Errorf("found context %s not the an expected phase (expectd=%s, found=%s)",
				build.Spec.Meta.Name,
				string(v1alpha1.IntegrationContextPhaseBuildRunning),
				string(target.Status.Phase),
			)
		}

		target.Status.Phase = v1alpha1.IntegrationContextPhaseBuildFailureRecovery

		if target.Status.Failure == nil {
			target.Status.Failure = &v1alpha1.Failure{
				Reason: build.Status.Error,
				Time:   metav1.Now(),
				Recovery: v1alpha1.FailureRecovery{
					Attempt:    0,
					AttemptMax: 5,
				},
			}
		}

		action.L.Error(fmt.Errorf(build.Status.Error), "IntegrationContext state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
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
		integration.Status.Phase = v1alpha1.IntegrationPhaseResolvingContext
		err = action.client.Status().Update(ctx, &integration)
		if err != nil {
			return err
		}
	}
	return nil
}
