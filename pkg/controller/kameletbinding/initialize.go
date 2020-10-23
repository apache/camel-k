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

package kameletbinding

import (
	"context"
	"encoding/json"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/bindings"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/patch"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// NewInitializeAction returns a action that initializes the kamelet binding configuration when not provided by the user
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(kameletbinding *v1alpha1.KameletBinding) bool {
	return kameletbinding.Status.Phase == v1alpha1.KameletBindingPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, kameletbinding *v1alpha1.KameletBinding) (*v1alpha1.KameletBinding, error) {
	controller := true
	blockOwnerDeletion := true
	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kameletbinding.Namespace,
			Name:      kameletbinding.Name,
			OwnerReferences: []metav1.OwnerReference{
				{
					APIVersion:         kameletbinding.APIVersion,
					Kind:               kameletbinding.Kind,
					Name:               kameletbinding.Name,
					UID:                kameletbinding.UID,
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			},
		},
	}
	// start from the integration spec defined in the binding
	if kameletbinding.Spec.Integration != nil {
		it.Spec = *kameletbinding.Spec.Integration.DeepCopy()
	}

	profile, err := action.determineProfile(ctx, kameletbinding)
	if err != nil {
		return nil, err
	}
	it.Spec.Profile = profile

	bindingContext := bindings.BindingContext{
		Ctx:       ctx,
		Client:    action.client,
		Namespace: it.Namespace,
		Profile:   profile,
	}

	from, err := bindings.Translate(bindingContext, v1alpha1.EndpointTypeSource, kameletbinding.Spec.Source)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine source URI")
	}
	to, err := bindings.Translate(bindingContext, v1alpha1.EndpointTypeSink, kameletbinding.Spec.Sink)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine sink URI")
	}

	if len(from.Traits) > 0 || len(to.Traits) > 0 {
		if it.Spec.Traits == nil {
			it.Spec.Traits = make(map[string]v1.TraitSpec)
		}
		for k, v := range from.Traits {
			it.Spec.Traits[k] = v
		}
		for k, v := range to.Traits {
			it.Spec.Traits[k] = v
		}
	}

	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": from.URI,
			"steps": []map[string]interface{}{
				{
					"to": to.URI,
				},
			},
		},
	}
	encodedFlow, err := json.Marshal(flow)
	if err != nil {
		return nil, err
	}
	it.Spec.Flows = append(it.Spec.Flows, v1.Flow{RawMessage: encodedFlow})

	if err := kubernetes.ReplaceResource(ctx, action.client, &it); err != nil {
		return nil, errors.Wrap(err, "could not create integration for kamelet binding")
	}

	// propagate Kamelet icon (best effort)
	action.propagateIcon(ctx, kameletbinding)

	target := kameletbinding.DeepCopy()
	target.Status.Phase = v1alpha1.KameletBindingPhaseCreating
	return target, nil
}

func (action *initializeAction) propagateIcon(ctx context.Context, binding *v1alpha1.KameletBinding) {
	icon, err := action.findIcon(ctx, binding)
	if err != nil {
		action.L.Errorf(err, "cannot find icon for kamelet binding %q", binding.Name)
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
	if _, ok := clone.Annotations[v1alpha1.AnnotationIcon]; !ok {
		clone.Annotations[v1alpha1.AnnotationIcon] = icon
	}
	p, err := patch.PositiveMergePatch(binding, clone)
	if err != nil {
		action.L.Errorf(err, "cannot compute patch to update icon for kamelet binding %q", binding.Name)
		return
	}
	if len(p) > 0 {
		if err := action.client.Patch(ctx, clone, client.RawPatch(types.MergePatchType, p)); err != nil {
			action.L.Errorf(err, "cannot apply merge patch to update icon for kamelet binding %q", binding.Name)
			return
		}
	}
}

func (action *initializeAction) findIcon(ctx context.Context, binding *v1alpha1.KameletBinding) (string, error) {
	var kameletRef *corev1.ObjectReference
	if binding.Spec.Source.Ref != nil && binding.Spec.Source.Ref.Kind == "Kamelet" && strings.HasPrefix(binding.Spec.Source.Ref.APIVersion, "camel.apache.org/") {
		kameletRef = binding.Spec.Source.Ref
	} else if binding.Spec.Sink.Ref != nil && binding.Spec.Sink.Ref.Kind == "Kamelet" && strings.HasPrefix(binding.Spec.Sink.Ref.APIVersion, "camel.apache.org/") {
		kameletRef = binding.Spec.Sink.Ref
	}

	if kameletRef == nil {
		return "", nil
	}

	key := client.ObjectKey{
		Namespace: binding.Namespace,
		Name:      kameletRef.Name,
	}
	var kamelet v1alpha1.Kamelet
	if err := action.client.Get(ctx, key, &kamelet); err != nil {
		return "", err
	}
	return kamelet.Annotations[v1alpha1.AnnotationIcon], nil
}

func (action *initializeAction) determineProfile(ctx context.Context, binding *v1alpha1.KameletBinding) (v1.TraitProfile, error) {
	if binding.Spec.Integration != nil && binding.Spec.Integration.Profile != "" {
		return binding.Spec.Integration.Profile, nil
	}
	pl, err := platform.GetCurrentPlatform(ctx, action.client, binding.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return "", errors.Wrap(err, "error while retrieving the integration platform")
	}
	if pl != nil {
		if pl.Status.Profile != "" {
			return pl.Status.Profile, nil
		}
		if pl.Spec.Profile != "" {
			return pl.Spec.Profile, nil
		}
	}
	if knative.IsEnabledInNamespace(ctx, action.client, binding.Namespace) {
		return v1.TraitProfileKnative, nil
	}
	if pl != nil {
		// Determine profile from cluster type
		plProfile := platform.GetProfile(pl)
		if plProfile != "" {
			return plProfile, nil
		}
	}
	return v1.DefaultTraitProfile, nil
}
