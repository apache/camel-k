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

package pipe

import (
	"context"
	"fmt"
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/kamelet/repository"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	"github.com/apache/camel-k/v2/pkg/util/patch"

	"github.com/apache/camel-k/v2/pkg/client"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewInitializeAction returns a action that initializes the Pipe configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(pipe *v1.Pipe) bool {
	return pipe.Status.Phase == v1.PipePhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, pipe *v1.Pipe) (*v1.Pipe, error) {
	action.L.Info("Initializing Pipe")

	return initializePipe(ctx, action.client, action.L, pipe)
}

func initializePipe(ctx context.Context, c client.Client, l log.Logger, pipe *v1.Pipe) (*v1.Pipe, error) {
	// Remove the previous conditions
	pipe.Status = v1.PipeStatus{}
	it, err := CreateIntegrationFor(ctx, c, pipe)
	if err != nil {
		pipe.Status.Phase = v1.PipePhaseError
		pipe.Status.SetErrorCondition(
			v1.PipeConditionReady,
			"IntegrationError",
			err,
		)

		return pipe, err
	}
	if _, err := kubernetes.ReplaceResource(ctx, c, it); err != nil {
		return nil, fmt.Errorf("could not create integration for Pipe: %w", err)
	}

	// propagate Kamelet icon (best effort)
	propagateIcon(ctx, c, l, pipe)

	target := pipe.DeepCopy()
	target.Status.Phase = v1.PipePhaseCreating

	return target, nil
}

func propagateIcon(ctx context.Context, c client.Client, l log.Logger, pipe *v1.Pipe) {
	icon, err := findIcon(ctx, c, pipe)
	if err != nil {
		l.Errorf(err, "some error happened while finding icon annotation for Pipe %q", pipe.Name)

		return
	}
	if icon == "" {
		return
	}

	// We must patch this here as we're changing the resource annotations and not the resource status
	err = patchPipeIconAnnotations(ctx, c, pipe, icon)
	l.Errorf(err, "some error happened while patching icon annotation for Pipe %q", pipe.Name)
}

func findIcon(ctx context.Context, c client.Client, pipe *v1.Pipe) (string, error) {
	var kameletRef *corev1.ObjectReference
	if pipe.Spec.Source.Ref != nil && pipe.Spec.Source.Ref.Kind == "Kamelet" && strings.HasPrefix(pipe.Spec.Source.Ref.APIVersion, "camel.apache.org/") {
		kameletRef = pipe.Spec.Source.Ref
	} else if pipe.Spec.Sink.Ref != nil && pipe.Spec.Sink.Ref.Kind == "Kamelet" && strings.HasPrefix(pipe.Spec.Sink.Ref.APIVersion, "camel.apache.org/") {
		kameletRef = pipe.Spec.Sink.Ref
	}

	if kameletRef == nil {
		return "", nil
	}

	repo, err := repository.New(ctx, c, pipe.Namespace, platform.GetOperatorNamespace())
	if err != nil {
		return "", err
	}

	kamelet, err := repo.Get(ctx, kameletRef.Name)
	if err != nil {
		return "", err
	}
	if kamelet == nil {
		return "", nil
	}

	return kamelet.Annotations[v1.AnnotationIcon], nil
}

func patchPipeIconAnnotations(ctx context.Context, c client.Client, pipe *v1.Pipe, icon string) error {
	clone := pipe.DeepCopy()
	clone.Annotations = make(map[string]string)
	for k, v := range pipe.Annotations {
		clone.Annotations[k] = v
	}
	if _, ok := clone.Annotations[v1.AnnotationIcon]; !ok {
		clone.Annotations[v1.AnnotationIcon] = icon
	}
	p, err := patch.MergePatch(pipe, clone)
	if err != nil {
		return err
	}
	if len(p) > 0 {
		if err := c.Patch(ctx, clone, ctrl.RawPatch(types.MergePatchType, p)); err != nil {
			return err
		}
	}

	return nil
}
