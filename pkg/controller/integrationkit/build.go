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
	"strings"
	"time"

	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/pkg/errors"

	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

// NewBuildAction creates a new build request handling action for the kit.
func NewBuildAction() Action {
	return &buildAction{}
}

type buildAction struct {
	baseAction
}

func (action *buildAction) Name() string {
	return "build"
}

func (action *buildAction) CanHandle(kit *v1.IntegrationKit) bool {
	return kit.Status.Phase == v1.IntegrationKitPhaseBuildSubmitted ||
		kit.Status.Phase == v1.IntegrationKitPhaseBuildRunning ||
		kit.Status.Phase == v1.IntegrationKitPhaseWaitingForCatalog
}

func (action *buildAction) Handle(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
	if kit.Status.Phase == v1.IntegrationKitPhaseBuildSubmitted ||
		kit.Status.Phase == v1.IntegrationKitPhaseWaitingForCatalog {
		return action.handleBuildSubmitted(ctx, kit)
	} else if kit.Status.Phase == v1.IntegrationKitPhaseBuildRunning {
		return action.handleBuildRunning(ctx, kit)
	}

	return nil, nil
}

func (action *buildAction) handleBuildSubmitted(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
	build, err := kubernetes.GetBuild(ctx, action.client, kit.Name, kit.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if err != nil && k8serrors.IsNotFound(err) ||
		build.Status.Phase == v1.BuildPhaseError ||
		build.Status.Phase == v1.BuildPhaseInterrupted ||
		build.Status.Phase == v1.BuildPhaseSucceeded {

		env, err := trait.Apply(ctx, action.client, nil, kit)
		if err != nil {
			return nil, err
		}

		if env.CamelCatalog == nil {
			return nil, errors.New("undefined camel catalog")
		}

		labels := kubernetes.FilterCamelCreatorLabels(kit.Labels)
		labels[v1.IntegrationKitLayoutLabel] = kit.Labels[v1.IntegrationKitLayoutLabel]

		annotations := make(map[string]string)
		if v, ok := kit.Annotations[v1.PlatformSelectorAnnotation]; ok {
			annotations[v1.PlatformSelectorAnnotation] = v
		}
		operatorID := defaults.OperatorID()
		if operatorID != "" {
			annotations[v1.OperatorIDAnnotation] = operatorID
		}

		timeout := env.Platform.Status.Build.GetTimeout()
		if layout := labels[v1.IntegrationKitLayoutLabel]; env.Platform.Spec.Build.Timeout == nil && layout == v1.IntegrationKitLayoutNative {
			// Increase the timeout to a sensible default
			timeout = metav1.Duration{
				Duration: 10 * time.Minute,
			}
		}

		// It has to be the same namespace as the operator as they must share a PVC
		builderPodNamespace := platform.GetOperatorNamespace()
		buildStrategy := env.Platform.Status.Build.BuildStrategy
		if env.BuildStrategy != "" {
			buildStrategy = env.BuildStrategy
		}

		// nolint: contextcheck
		if buildStrategy == v1.BuildStrategyPod {
			// Pod strategy requires a PVC to exist. If it does not exist, we warn the user and fallback to Routine build strategy
			if pvc, err := kubernetes.LookupPersistentVolumeClaim(env.Ctx, env.Client, builderPodNamespace, defaults.DefaultPVC); pvc != nil || err != nil {
				err = platform.CreateBuilderServiceAccount(env.Ctx, env.Client, env.Platform)
				if err != nil {
					return nil, errors.Wrap(err, "Error while creating Camel K Builder service account")
				}
			} else {
				// Fallback to Routine strategy
				buildStrategy = v1.BuildStrategyRoutine
				Log.Info(`Warning: the operator was installed with an ephemeral storage, builder "pod" strategy is not supported: using "routine" build strategy as a fallback. We recommend to configure a PersistentVolumeClaim in order to be able to use "pod" builder strategy. Please consider that certain features such as Quarkus native require a "pod" builder strategy (hence a PVC) to work properly.`)
			}
		}

		build = &v1.Build{
			TypeMeta: metav1.TypeMeta{
				APIVersion: v1.SchemeGroupVersion.String(),
				Kind:       v1.BuildKind,
			},
			ObjectMeta: metav1.ObjectMeta{
				Namespace:   kit.Namespace,
				Name:        kit.Name,
				Labels:      labels,
				Annotations: annotations,
			},
			Spec: v1.BuildSpec{
				Strategy:            buildStrategy,
				ToolImage:           env.CamelCatalog.Image,
				BuilderPodNamespace: builderPodNamespace,
				Tasks:               env.BuildTasks,
				Timeout:             timeout,
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

	if build.Status.Phase == v1.BuildPhaseRunning {
		kit.Status.Phase = v1.IntegrationKitPhaseBuildRunning
		return kit, nil
	}

	return nil, nil
}

func (action *buildAction) handleBuildRunning(ctx context.Context, kit *v1.IntegrationKit) (*v1.IntegrationKit, error) {
	build, err := kubernetes.GetBuild(ctx, action.client, kit.Name, kit.Namespace)
	if err != nil {
		return nil, err
	}

	switch build.Status.Phase {
	case v1.BuildPhaseRunning:
		action.L.Info("Build running")
	case v1.BuildPhaseSucceeded:
		// we should ensure that the integration kit is still in the right phase,
		// if not there is a chance that the kit has been modified by the user
		if kit.Status.Phase != v1.IntegrationKitPhaseBuildRunning {
			return nil, fmt.Errorf("found kit %s not in the expected phase (expectd=%s, found=%s)",
				kit.Name,
				string(v1.IntegrationKitPhaseBuildRunning),
				string(kit.Status.Phase),
			)
		}

		kit.Status.BaseImage = build.Status.BaseImage
		kit.Status.Image = build.Status.Image

		// Address the image by repository digest instead of tag if possible
		if build.Status.Digest != "" {
			image := kit.Status.Image
			i := strings.LastIndex(image, ":")
			if i > 0 {
				image = image[:i]
			}
			kit.Status.Image = fmt.Sprintf("%s@%s", image, build.Status.Digest)
		} else {
			// otherwise rely on repository tag
			kit.Status.Image = build.Status.Image
		}

		kit.Status.Phase = v1.IntegrationKitPhaseReady
		kit.Status.Artifacts = make([]v1.Artifact, 0, len(build.Status.Artifacts))

		for _, a := range build.Status.Artifacts {
			// do not include artifact location
			kit.Status.Artifacts = append(kit.Status.Artifacts, v1.Artifact{
				ID:       a.ID,
				Location: "",
				Target:   a.Target,
				Checksum: a.Checksum,
			})
		}

		return kit, err
	case v1.BuildPhaseError, v1.BuildPhaseInterrupted:
		// we should ensure that the integration kit is still in the right phase,
		// if not there is a chance that the kit has been modified by the user
		if kit.Status.Phase != v1.IntegrationKitPhaseBuildRunning {
			return nil, fmt.Errorf("found kit %s not the an expected phase (expectd=%s, found=%s)",
				kit.Name,
				string(v1.IntegrationKitPhaseBuildRunning),
				string(kit.Status.Phase),
			)
		}

		// Let's copy the build failure to the integration kit status
		kit.Status.Failure = build.Status.Failure
		kit.Status.Phase = v1.IntegrationKitPhaseError

		return kit, nil
	}

	return nil, nil
}
