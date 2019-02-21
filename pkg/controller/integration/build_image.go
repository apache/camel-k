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

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/apache/camel-k/pkg/util/cancellable"

	"github.com/apache/camel-k/pkg/util/source"

	"github.com/pkg/errors"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/digest"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	corev1 "k8s.io/api/core/v1"
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
	if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageSubmitted {
		return true
	}
	if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageRunning {
		return true
	}

	return false
}

func (action *buildImageAction) Handle(ctx context.Context, integration *v1alpha1.Integration) error {
	if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageSubmitted {
		return action.handleBuildImageSubmitted(ctx, integration)
	}
	if integration.Status.Phase == v1alpha1.IntegrationPhaseBuildImageRunning {
		return action.handleBuildImageRunning(ctx, integration)
	}

	return nil
}

func (action *buildImageAction) handleBuildImageRunning(_ context.Context, integration *v1alpha1.Integration) error {
	b, err := platform.GetPlatformBuilder(action.client, integration.Namespace)
	if err != nil {
		return err
	}

	if b.IsBuilding(integration.ObjectMeta) {
		action.L.Info("Build running")
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

	b, err := platform.GetPlatformBuilder(action.client, integration.Namespace)
	if err != nil {
		return err
	}

	if !b.IsBuilding(ictx.ObjectMeta) {
		env, err := trait.Apply(ctx, action.client, integration, ictx)
		if err != nil {
			return err
		}
		if env.CamelCatalog == nil {
			return errors.New("undefined camel catalog")
		}

		// This build do not require to determine dependencies nor a project, the
		// builder step do remove them
		//
		// the context given to the handler is per reconcile loop and as the build
		// happens asynchronously, a new context has to be created. the new context
		// can be used also to stop the build.
		r := builder.Request{
			C:              cancellable.NewContext(),
			Catalog:        env.CamelCatalog,
			RuntimeVersion: env.RuntimeVersion,
			Meta:           integration.ObjectMeta,
			Steps:          env.Steps,
			BuildDir:       env.BuildDir,
			Platform:       env.Platform.Spec,
			Image:          ictx.Status.Image,
			// Sources are added as part of the standard deployment bits
			Resources: make([]builder.Resource, 0, len(integration.Spec.Sources)),
		}

		// inline resources so they are copied over the generated
		// container image
		if err := action.inlineResources(ctx, integration, &r, env); err != nil {
			return err
		}

		b.Submit(r, func(result *builder.Result) {
			//
			// this function is invoked synchronously for every state change
			//
			if err := action.handleBuildStateChange(result.Request.C, result); err != nil {
				action.L.Error(err, "Error while building integration image")
			}
		})
	}

	return nil
}

func (action *buildImageAction) handleBuildStateChange(ctx context.Context, res *builder.Result) error {
	//
	// Get the latest status of the integration
	//
	target, err := kubernetes.GetIntegration(ctx, action.client, res.Request.Meta.Name, res.Request.Meta.Namespace)
	if err != nil || target == nil {
		return err
	}

	switch res.Status {
	case builder.StatusSubmitted:
		action.L.Info("Build submitted")
	case builder.StatusStarted:
		target.Status.Phase = v1alpha1.IntegrationPhaseBuildImageRunning

		action.L.Info("Integration state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
	case builder.StatusError:
		target.Status.Phase = v1alpha1.IntegrationPhaseBuildFailureRecovery

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

		action.L.Error(res.Error, "Integration state transition", "phase", target.Status.Phase)

		return action.client.Status().Update(ctx, target)
	case builder.StatusCompleted:
		target.Status.Phase = v1alpha1.IntegrationPhaseDeploying
		if res.PublicImage != "" {
			target.Status.Image = res.PublicImage
		} else {
			target.Status.Image = res.Image
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
	}

	return nil
}

func (action *buildImageAction) inlineResources(ctx context.Context, integration *v1alpha1.Integration, r *builder.Request, e *trait.Environment) error {
	sources, err := source.Resolve(integration.Sources(), func(name string) (*corev1.ConfigMap, error) {
		cm := e.Resources.GetConfigMap(func(cm *corev1.ConfigMap) bool {
			return cm.Name == name
		})

		if cm != nil {
			return cm, nil
		}

		return kubernetes.GetConfigMap(ctx, action.client, name, integration.Namespace)
	})

	if err != nil {
		return err
	}

	for _, data := range sources {
		r.Resources = append(r.Resources, builder.Resource{
			Content: []byte(data.Content),
			Target:  path.Join("sources", data.Name),
		})
	}

	return nil
}
