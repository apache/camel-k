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
	"github.com/apache/camel-k/v2/pkg/util/patch"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
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

func (action *initializeAction) CanHandle(binding *v1.Pipe) bool {
	return binding.Status.Phase == v1.PipePhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, binding *v1.Pipe) (*v1.Pipe, error) {
	action.L.Info("Initializing Pipe")

	if binding.Spec.Integration != nil {
		action.L.Infof("Pipe %s is using deprecated .spec.integration parameter. Please, update and use annotation traits instead", binding.Name)
		binding.Status.SetCondition(
			v1.PipeIntegrationDeprecationNotice,
			corev1.ConditionTrue,
			".spec.integration parameter is deprecated",
			".spec.integration parameter is deprecated. Use annotation traits instead",
		)
	}
	it, err := CreateIntegrationFor(ctx, action.client, binding)
	if err != nil {
		binding.Status.Phase = v1.PipePhaseError
		binding.Status.SetErrorCondition(v1.PipeIntegrationConditionError,
			"Couldn't create an Integration custom resource", err)
		return binding, err
	}

	if _, err := kubernetes.ReplaceResource(ctx, action.client, it); err != nil {
		return nil, fmt.Errorf("could not create integration for Pipe: %w", err)
	}

	// propagate Kamelet icon (best effort)
	action.propagateIcon(ctx, binding)

	target := binding.DeepCopy()
	target.Status.Phase = v1.PipePhaseCreating
	return target, nil
}

func (action *initializeAction) propagateIcon(ctx context.Context, binding *v1.Pipe) {
	icon, err := action.findIcon(ctx, binding)
	if err != nil {
		action.L.Errorf(err, "cannot find icon for Pipe %q", binding.Name)
		return
	}
	if icon == "" {
		return
	}
	// compute patch
	clone := binding.DeepCopy()
	clone.Annotations = make(map[string]string)
	for k, v := range binding.Annotations {
		clone.Annotations[k] = v
	}
	if _, ok := clone.Annotations[v1.AnnotationIcon]; !ok {
		clone.Annotations[v1.AnnotationIcon] = icon
	}
	p, err := patch.MergePatch(binding, clone)
	if err != nil {
		action.L.Errorf(err, "cannot compute patch to update icon for Binding %q", binding.Name)
		return
	}
	if len(p) > 0 {
		if err := action.client.Patch(ctx, clone, client.RawPatch(types.MergePatchType, p)); err != nil {
			action.L.Errorf(err, "cannot apply merge patch to update icon for Pipe %q", binding.Name)
			return
		}
	}
}

func (action *initializeAction) findIcon(ctx context.Context, binding *v1.Pipe) (string, error) {
	var kameletRef *corev1.ObjectReference
	if binding.Spec.Source.Ref != nil && binding.Spec.Source.Ref.Kind == "Kamelet" && strings.HasPrefix(binding.Spec.Source.Ref.APIVersion, "camel.apache.org/") {
		kameletRef = binding.Spec.Source.Ref
	} else if binding.Spec.Sink.Ref != nil && binding.Spec.Sink.Ref.Kind == "Kamelet" && strings.HasPrefix(binding.Spec.Sink.Ref.APIVersion, "camel.apache.org/") {
		kameletRef = binding.Spec.Sink.Ref
	}

	if kameletRef == nil {
		return "", nil
	}

	repo, err := repository.New(ctx, action.client, binding.Namespace, platform.GetOperatorNamespace())
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
