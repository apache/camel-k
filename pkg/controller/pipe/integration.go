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
	"encoding/json"
	"errors"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"

	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/util"
	"github.com/apache/camel-k/v2/pkg/util/bindings"
	"github.com/apache/camel-k/v2/pkg/util/knative"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/property"
)

var (
	endpointTypeSourceContext = bindings.EndpointContext{Type: v1.EndpointTypeSource}
	endpointTypeSinkContext   = bindings.EndpointContext{Type: v1.EndpointTypeSink}
)

// CreateIntegrationFor creates and Integration from a Pipe.
func CreateIntegrationFor(ctx context.Context, c client.Client, pipe *v1.Pipe) (*v1.Integration, error) {
	controller := true
	blockOwnerDeletion := true
	annotations := util.CopyMap(pipe.Annotations)
	// avoid propagating the icon to the integration as it's heavyweight and not needed
	delete(annotations, v1.AnnotationIcon)
	traits := pipe.Spec.Traits

	it := v1.Integration{
		Namespace:   pipe.Namespace,
		Name:        pipe.Name,
		Annotations: annotations,
		Labels:      util.CopyMap(pipe.Labels),
		OwnerReferences: []metav1.OwnerReference{
			{
				APIVersion:         pipe.APIVersion,
				Kind:               pipe.Kind,
				Name:               pipe.Name,
				UID:                pipe.UID,
				Controller:         &controller,
				BlockOwnerDeletion: &blockOwnerDeletion,
			},
		},
	}

	// creator labels
	if it.GetLabels() == nil {
		it.SetLabels(make(map[string]string))
	}
	it.GetLabels()[kubernetes.CamelCreatorLabelKind] = pipe.Kind
	it.GetLabels()[kubernetes.CamelCreatorLabelName] = pipe.Name

	if traits != nil {
		it.Spec.Traits = *traits
	}

	if pipe.Spec.Dependencies != nil {
		it.Spec.Dependencies = pipe.Spec.Dependencies
	}

	// Set replicas (or override podspecable value) if present
	if pipe.Spec.Replicas != nil {
		replicas := *pipe.Spec.Replicas
		it.Spec.Replicas = &replicas
	}

	profile, err := determineTraitProfile(c)
	if err != nil {
		return nil, err
	}
	//nolint:staticcheck
	it.Spec.Profile = profile

	if pipe.Spec.ServiceAccountName != "" {
		it.Spec.ServiceAccountName = pipe.Spec.ServiceAccountName
	}

	bindingContext := bindings.BindingContext{
		Ctx:                ctx,
		Client:             c,
		Namespace:          it.Namespace,
		Profile:            profile,
		Metadata:           it.Annotations,
		ServiceAccountName: it.Spec.ServiceAccountName,
	}

	from, err := bindings.Translate(bindingContext, endpointTypeSourceContext, pipe.Spec.Source)
	if err != nil {
		return nil, err
	}
	to, err := bindings.Translate(bindingContext, endpointTypeSinkContext, pipe.Spec.Sink)
	if err != nil {
		return nil, err
	}
	// error handler is optional
	errorHandler, err := maybeErrorHandler(pipe.Spec.ErrorHandler, bindingContext)
	if err != nil {
		return nil, err
	}

	steps := make([]*bindings.Binding, 0, len(pipe.Spec.Steps))
	for idx, step := range pipe.Spec.Steps {
		position := idx
		stepBinding, err := bindings.Translate(bindingContext, bindings.EndpointContext{
			Type:     v1.EndpointTypeAction,
			Position: &position,
		}, step)
		if err != nil {
			return nil, fmt.Errorf("could not determine URI for step %d: %w", idx, err)
		}
		steps = append(steps, stepBinding)
	}

	if to.Step == nil && to.URI == "" {
		return nil, errors.New("illegal step definition for sink step: either Step or URI should be provided")
	}
	if from.URI == "" {
		return nil, errors.New("illegal step definition for source step: URI should be provided")
	}
	for index, step := range steps {
		if step.Step == nil && step.URI == "" {
			return nil, fmt.Errorf("illegal step definition for step %d: either Step or URI should be provided", index)
		}
	}

	if err := configureBinding(&it, from); err != nil {
		return nil, err
	}

	if err := configureBinding(&it, steps...); err != nil {
		return nil, err
	}

	if err := configureBinding(&it, to); err != nil {
		return nil, err
	}

	if err := configureBinding(&it, errorHandler); err != nil {
		return nil, err
	}

	if it.Spec.Configuration != nil {
		sort.SliceStable(it.Spec.Configuration, func(i, j int) bool {
			mi, mj := it.Spec.Configuration[i], it.Spec.Configuration[j]
			switch {
			case mi.Type != mj.Type:
				return mi.Type < mj.Type
			default:
				return mi.Value < mj.Value
			}
		})
	}

	dslSteps := make([]map[string]any, 0)

	if from.Step != nil {
		dslSteps = append(dslSteps, from.AsYamlDSL())
	}

	for _, step := range steps {
		dslSteps = append(dslSteps, step.AsYamlDSL())
	}

	if to.Step != nil {
		dslSteps = append(dslSteps, to.AsYamlDSL())
	}
	dslSteps = append(dslSteps, map[string]any{
		"to": to.URI,
	})

	fromWrapper := map[string]any{
		"uri":   from.URI,
		"steps": dslSteps,
	}

	flowRoute := map[string]any{
		"route": map[string]any{
			"id":   "binding",
			"from": fromWrapper,
		},
	}

	if errorHandler != nil {
		eh := translateCamelErrorHandler(errorHandler)
		encodedErrorHandler, err := json.Marshal(eh)
		if err != nil {
			return nil, err
		}
		it.Spec.Flows = append(it.Spec.Flows, v1.Flow{RawMessage: encodedErrorHandler})
	}

	encodedRoute, err := json.Marshal(flowRoute)
	if err != nil {
		return nil, err
	}

	it.Spec.Flows = append(it.Spec.Flows, v1.Flow{RawMessage: encodedRoute})

	return &it, nil
}

func configureBinding(integration *v1.Integration, bindings ...*bindings.Binding) error {
	for _, b := range bindings {
		if b == nil {
			continue
		}
		//nolint:staticcheck
		if err := integration.Spec.Traits.Merge(b.Traits); err != nil {
			return err
		}
		for k, v := range b.ApplicationProperties {
			entry, err := property.EncodePropertyFileEntry(k, v)
			if err != nil {
				return err
			}

			integration.Spec.AddConfigurationProperty(entry)
		}
	}

	return nil
}

//nolint:staticcheck
func determineTraitProfile(c client.Client) (v1.TraitProfile, error) {
	if ok, err := knative.IsInstalled(c); err != nil {
		return "", err
	} else if ok {
		return v1.TraitProfileKnative, nil
	}

	return v1.DefaultTraitProfile, nil
}
