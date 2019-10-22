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

package integrationkit

import (
	"context"
	"fmt"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/kubernetes"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/pkg/errors"
)

// NewBuildAction creates a new build request handling action for the kit
func NewBuildAction() Action {
	return &buildAction{}
}

type buildAction struct {
	baseAction
}

func (action *buildAction) Name() string {
	return "build-submitted"
}

func (action *buildAction) CanHandle(kit *v1alpha1.IntegrationKit) bool {
	return kit.Status.Phase == v1alpha1.IntegrationKitPhaseBuildSubmitted ||
		kit.Status.Phase == v1alpha1.IntegrationKitPhaseBuildRunning
}

func (action *buildAction) Handle(ctx context.Context, kit *v1alpha1.IntegrationKit) (*v1alpha1.IntegrationKit, error) {
	if kit.Status.Phase == v1alpha1.IntegrationKitPhaseBuildSubmitted {
		return action.handleBuildSubmitted(ctx, kit)
	} else if kit.Status.Phase == v1alpha1.IntegrationKitPhaseBuildRunning {
		return action.handleBuildRunning(ctx, kit)
	}

	return nil, nil
}

func (action *buildAction) handleBuildSubmitted(ctx context.Context, kit *v1alpha1.IntegrationKit) (*v1alpha1.IntegrationKit, error) {
	build, err := kubernetes.GetBuild(ctx, action.client, kit.Name, kit.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8serrors.IsNotFound(err) ||
		build.Status.Phase == v1alpha1.BuildPhaseError ||
		build.Status.Phase == v1alpha1.BuildPhaseInterrupted ||
		build.Status.Phase == v1alpha1.BuildPhaseSucceeded {

		env, err := trait.Apply(ctx, action.client, nil, kit)
		if err != nil {
			return nil, err
		}

		if env.CamelCatalog == nil {
			return nil, errors.New("undefined camel catalog")
		}

		build = &v1alpha1.Build{
			TypeMeta: metav1.TypeMeta{
				APIVersion: "camel.apache.org/v1alpha1",
				Kind:       "Build",
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace: kit.Namespace,
				Name:      kit.Name,
			},
			Spec: v1alpha1.BuildSpec{
				Meta:            kit.ObjectMeta,
				CamelVersion:    env.CamelCatalog.Version,
				RuntimeVersion:  env.RuntimeVersion,
				RuntimeProvider: env.CamelCatalog.RuntimeProvider,
				Platform:        env.Platform.Spec,
				Dependencies:    kit.Spec.Dependencies,
				// TODO: sort for easy read
				Steps:    builder.StepIDsFor(env.Steps...),
				BuildDir: env.BuildDir,
			},
		}

		// Set the integration kit instance as the owner and controller
		if err := controllerutil.SetControllerReference(kit, build, action.client.GetScheme()); err != nil {
			return nil, err
		}

		err = action.client.Delete(ctx, build)
		if err != nil && !k8serrors.IsNotFound(err) {
			return nil, errors.Wrap(err, "cannot delete build")
		}

		err = action.client.Create(ctx, build)
		if err != nil {
			return nil, errors.Wrap(err, "cannot create build")
		}
	}

	if build.Status.Phase == v1alpha1.BuildPhaseRunning {
		kit.Status.Phase = v1alpha1.IntegrationKitPhaseBuildRunning
		return kit, nil
	}

	return nil, nil
}

func (action *buildAction) handleBuildRunning(ctx context.Context, kit *v1alpha1.IntegrationKit) (*v1alpha1.IntegrationKit, error) {
	build, err := kubernetes.GetBuild(ctx, action.client, kit.Name, kit.Namespace)
	if err != nil {
		return nil, err
	}

	switch build.Status.Phase {
	case v1alpha1.BuildPhaseRunning:
		action.L.Info("Build running")
	case v1alpha1.BuildPhaseSucceeded:
		// we should ensure that the integration kit is still in the right phase,
		// if not there is a chance that the kit has been modified by the user
		if kit.Status.Phase != v1alpha1.IntegrationKitPhaseBuildRunning {
			return nil, fmt.Errorf("found kit %s not in the expected phase (expectd=%s, found=%s)",
				build.Spec.Meta.Name,
				string(v1alpha1.IntegrationKitPhaseBuildRunning),
				string(kit.Status.Phase),
			)
		}

		kit.Status.BaseImage = build.Status.BaseImage
		kit.Status.Image = build.Status.Image
		kit.Status.Phase = v1alpha1.IntegrationKitPhaseReady
		kit.Status.Artifacts = make([]v1alpha1.Artifact, 0, len(build.Status.Artifacts))

		for _, a := range build.Status.Artifacts {
			// do not include artifact location
			kit.Status.Artifacts = append(kit.Status.Artifacts, v1alpha1.Artifact{
				ID:       a.ID,
				Location: "",
				Target:   a.Target,
			})
		}

		return kit, err
	case v1alpha1.BuildPhaseError, v1alpha1.BuildPhaseInterrupted:
		// we should ensure that the integration kit is still in the right phase,
		// if not there is a chance that the kit has been modified by the user
		if kit.Status.Phase != v1alpha1.IntegrationKitPhaseBuildRunning {
			return nil, fmt.Errorf("found kit %s not the an expected phase (expectd=%s, found=%s)",
				build.Spec.Meta.Name,
				string(v1alpha1.IntegrationKitPhaseBuildRunning),
				string(kit.Status.Phase),
			)
		}

		// Let's copy the build failure to the integration kit status
		kit.Status.Failure = build.Status.Failure
		kit.Status.Phase = v1alpha1.IntegrationKitPhaseError

		return kit, nil
	}

	return nil, nil
}
