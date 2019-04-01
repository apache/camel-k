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

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// NewBuildImageAction create an action that handles integration image build
func NewBuildImageAction() Action {
	return &buildImageAction{}
}

type buildImageAction struct {
	baseAction
}

func (action *buildImageAction) Name() string {
	return "build-image"
}

func (action *buildImageAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageSubmitted ||
		integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageRunning
}

func (action *buildImageAction) Handle(ctx context.Context, integration *v1alpha1.Integration) error {
	if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageSubmitted {
		return action.handleBuildImageSubmitted(ctx, integration)
	} else if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageRunning {
		return action.handleBuildImageRunning(ctx, integration)
	}

	return nil
}

func (action *buildImageAction) handleBuildImageSubmitted(ctx context.Context, integration *v1alpha1.Integration) error {
	// in this phase the integration need to be associated to a context whose image
	// will be used as base image for the integration images
	if integration.Status.Context == "" {
		return fmt.Errorf("context is not set for integration: %s", integration.Name)
	}

	// look-up the integration context associated to this integration, this is needed
	// to determine the base image
	ictx, err := kubernetes.GetIntegrationContext(ctx, action.client, integration.Status.Context, integration.Namespace)
	if err != nil || ictx == nil {
		return errors.Wrapf(err, "unable to find integration context %s, %s", integration.Status.Context, err)
	}

	build := &v1alpha1.Build{}
	err = action.client.Get(ctx, types.NamespacedName{Namespace: integration.Namespace, Name: integration.Name}, build)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	if err != nil && k8serrors.IsNotFound(err) {
		env, err := trait.Apply(ctx, action.client, integration, ictx)
		if err != nil {
			return err
		}
		if env.CamelCatalog == nil {
			return errors.New("undefined camel catalog")
		}

		// This build do not require to determine dependencies nor a project, the
		// builder step do remove them

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
				Namespace: integration.Namespace,
				Name:      integration.Name,
			},
			Spec: v1alpha1.BuildSpec{
				Meta:           integration.ObjectMeta,
				Image:          ictx.Status.Image,
				CamelVersion:   env.CamelCatalog.Version,
				RuntimeVersion: env.RuntimeVersion,
				Platform:       env.Platform.Spec,
				Steps:          steps,
				BuildDir:       env.BuildDir,
			},
		}

		// Inline source and resources so they are copied over the generated
		// container image. For the time being, references are being resolved
		// and their content serialized. We may want to resolve after the build
		// is submitted, just before the build is run.
		if err := action.inlineSources(ctx, integration, build, env); err != nil {
			return err
		}
		if err := action.inlineResources(ctx, integration, build, env); err != nil {
			return err
		}

		// Set the integration context instance as the owner and controller
		if err := controllerutil.SetControllerReference(integration, build, action.client.GetScheme()); err != nil {
			return err
		}

		err = action.client.Delete(ctx, build)
		if err != nil && !k8serrors.IsNotFound(err) {
			return errors.Wrap(err, "cannot delete build")
		}

		err = action.client.Create(ctx, build)
		if err != nil {
			return errors.Wrap(err, "cannot create build")
		}
	}

	if build.Status.Phase == v1alpha1.BuildPhaseRunning {
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseBuildImageRunning

		action.L.Info("Integration state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
	}

	return nil
}

func (action *buildImageAction) handleBuildImageRunning(ctx context.Context, integration *v1alpha1.Integration) error {
	build := &v1alpha1.Build{}
	err := action.client.Get(ctx, types.NamespacedName{Namespace: integration.Namespace, Name: integration.Name}, build)
	if err != nil && !k8serrors.IsNotFound(err) {
		return err
	}

	switch build.Status.Phase {

	case v1alpha1.BuildPhaseRunning:
		action.L.Info("Build running")

	case v1alpha1.BuildPhaseSucceeded:
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
		if build.Status.PublicImage != "" {
			target.Status.Image = build.Status.PublicImage
		} else {
			target.Status.Image = build.Status.Image
		}

		dgst, err := digest.ComputeForIntegration(target)
		if err != nil {
			return err
		}

		target.Status.Digest = dgst

		action.L.Info("Integration state transition", "phase", target.Status.Phase)

		if err := action.client.Status().Update(ctx, target); err != nil {
			return err
		}

	case v1alpha1.BuildPhaseFailed, v1alpha1.BuildPhaseInterrupted:
		target := integration.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationPhaseBuildFailureRecovery

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

		action.L.Error(fmt.Errorf(build.Status.Error), "Integration state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
	}

	return nil
}

func (action *buildImageAction) inlineSources(ctx context.Context, integration *v1alpha1.Integration, build *v1alpha1.Build, e *trait.Environment) error {
	sources, err := kubernetes.ResolveIntegrationSources(ctx, action.client, integration, e.Resources)
	if err != nil {
		return err
	}

	build.Spec.Sources = make([]v1alpha1.SourceSpec, 0, len(sources))
	build.Spec.Sources = append(build.Spec.Sources, sources...)

	return nil
}

func (action *buildImageAction) inlineResources(ctx context.Context, integration *v1alpha1.Integration, build *v1alpha1.Build, e *trait.Environment) error {
	resources, err := kubernetes.ResolveIntegrationResources(ctx, action.client, integration, e.Resources)
	if err != nil {
		return err
	}

	build.Spec.Resources = make([]v1alpha1.ResourceSpec, 0, len(resources))
	build.Spec.Resources = append(build.Spec.Resources, resources...)

	return nil
}
