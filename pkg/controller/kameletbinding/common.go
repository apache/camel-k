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
	"fmt"
	"sort"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/bindings"
	"github.com/apache/camel-k/pkg/util/knative"
	"github.com/pkg/errors"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createIntegrationFor(ctx context.Context, c client.Client, kameletbinding *v1alpha1.KameletBinding) (*v1.Integration, error) {
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

	profile, err := determineProfile(ctx, c, kameletbinding)
	if err != nil {
		return nil, err
	}
	it.Spec.Profile = profile

	bindingContext := bindings.BindingContext{
		Ctx:       ctx,
		Client:    c,
		Namespace: it.Namespace,
		Profile:   profile,
	}

	from, err := bindings.Translate(bindingContext, bindings.EndpointContext{Type: v1alpha1.EndpointTypeSource}, kameletbinding.Spec.Source)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine source URI")
	}
	to, err := bindings.Translate(bindingContext, bindings.EndpointContext{Type: v1alpha1.EndpointTypeSink}, kameletbinding.Spec.Sink)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine sink URI")
	}
	var errorHandler *bindings.Binding
	if kameletbinding.Spec.ErrorHandler.Ref != nil || kameletbinding.Spec.ErrorHandler.URI != nil {
		errorHandler, err = bindings.Translate(bindingContext, bindings.EndpointContext{Type: v1alpha1.EndpointTypeErrorHandler}, kameletbinding.Spec.ErrorHandler)
		if err != nil {
			return nil, errors.Wrap(err, "could not determine error handler URI")
		}

		err = setErrorHandlerKamelet(errorHandler, kameletbinding.Spec.ErrorHandler)
		if err != nil {
			return nil, errors.Wrap(err, "could not set error handler")
		}
	}

	steps := make([]*bindings.Binding, 0, len(kameletbinding.Spec.Steps))
	for idx, step := range kameletbinding.Spec.Steps {
		position := idx
		stepBinding, err := bindings.Translate(bindingContext, bindings.EndpointContext{
			Type:     v1alpha1.EndpointTypeAction,
			Position: &position,
		}, step)
		if err != nil {
			return nil, errors.Wrapf(err, "could not determine URI for step %d", idx)
		}
		steps = append(steps, stepBinding)
	}

	allBindings := make([]*bindings.Binding, 0, len(steps)+3)
	allBindings = append(allBindings, from)
	allBindings = append(allBindings, steps...)
	allBindings = append(allBindings, to)
	if errorHandler != nil {
		allBindings = append(allBindings, errorHandler)
	}

	propList := make([]string, 0)
	for _, b := range allBindings {
		if it.Spec.Traits == nil {
			it.Spec.Traits = make(map[string]v1.TraitSpec)
		}
		for k, v := range b.Traits {
			it.Spec.Traits[k] = v
		}
		for k, v := range b.ApplicationProperties {
			propList = append(propList, fmt.Sprintf("%s=%s", k, v))
		}
	}

	sort.Strings(propList)
	for _, p := range propList {
		it.Spec.Configuration = append(it.Spec.Configuration, v1.ConfigurationSpec{
			Type:  "property",
			Value: p,
		})
	}

	dslSteps := make([]map[string]interface{}, 0)
	for _, step := range steps {
		dslSteps = append(dslSteps, map[string]interface{}{
			"to": step.URI,
		})
	}
	dslSteps = append(dslSteps, map[string]interface{}{
		"to": to.URI,
	})

	flowFrom := map[string]interface{}{
		"from": map[string]interface{}{
			"uri":   from.URI,
			"steps": dslSteps,
		},
	}
	encodedFrom, err := json.Marshal(flowFrom)
	if err != nil {
		return nil, err
	}
	it.Spec.Flows = append(it.Spec.Flows, v1.Flow{RawMessage: encodedFrom})

	return &it, nil
}

func setErrorHandlerKamelet(errorHandler *bindings.Binding, kameletSpec v1alpha1.Endpoint) error {
	if errorHandler.ApplicationProperties == nil {
		errorHandler.ApplicationProperties = make(map[string]string)
	}
	if kameletSpec.URI != nil {
		if !strings.HasPrefix(*kameletSpec.URI, "kamelet") {
			return fmt.Errorf("Kamelet Binding only supports kamelet as error handler, provided: %s", *kameletSpec.URI)
		}

		errorHandler.ApplicationProperties["camel.k.default-error-handler"] = *kameletSpec.URI
		return nil
	}

	errorHandler.ApplicationProperties["camel.k.default-error-handler"] = kameletSpec.Ref.Name
	return nil
}

func determineProfile(ctx context.Context, c client.Client, binding *v1alpha1.KameletBinding) (v1.TraitProfile, error) {
	if binding.Spec.Integration != nil && binding.Spec.Integration.Profile != "" {
		return binding.Spec.Integration.Profile, nil
	}
	pl, err := platform.GetCurrent(ctx, c, binding.Namespace)
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
	if knative.IsEnabledInNamespace(ctx, c, binding.Namespace) {
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
