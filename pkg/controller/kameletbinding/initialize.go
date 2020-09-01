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
	"net/url"
	"strings"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/apache/camel-k/pkg/util/uri"
	"github.com/pkg/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	it := v1.Integration{
		ObjectMeta: metav1.ObjectMeta{
			Namespace: kameletbinding.Namespace,
			Name:      kameletbinding.Name,
		},
	}
	// start from the integration spec defined in the binding
	if kameletbinding.Spec.Integration != nil {
		it.Spec = *kameletbinding.Spec.Integration.DeepCopy()
	}

	fromURI, err := getEndpointURI(kameletbinding.Spec.Source)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine source URI")
	}
	toURI, err := getEndpointURI(kameletbinding.Spec.Sink)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine sink URI")
	}

	// TODO remove this after making sinkbinding the default (https://github.com/apache/camel-k/issues/1654)
	if strings.HasPrefix(toURI, "knative:") {
		knativeConfig := map[string]interface{}{
			"sinkBinding": true,
		}
		knativeConfigJSON, err := json.Marshal(knativeConfig)
		if err != nil {
			return nil, err
		}
		if it.Spec.Traits == nil {
			it.Spec.Traits = make(map[string]v1.TraitSpec)
		}
		it.Spec.Traits["knative"] = v1.TraitSpec{
			Configuration: v1.TraitConfiguration{
				RawMessage: knativeConfigJSON,
			},
		}
	}

	flow := map[string]interface{}{
		"from": map[string]interface{}{
			"uri": fromURI,
			"steps": []map[string]interface{}{
				{
					"to": toURI,
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

	target := kameletbinding.DeepCopy()
	target.Status.Phase = v1alpha1.KameletBindingPhaseCreating
	return target, nil
}

func getEndpointURI(e v1alpha1.Endpoint) (string, error) {
	baseURI, err := getEndpointBaseURI(e)
	if err != nil {
		return baseURI, err
	}

	// Convert json properties to string before using them in URI
	if len(e.Properties.RawMessage) > 0 {
		var props map[string]interface{}
		if err := json.Unmarshal(e.Properties.RawMessage, &props); err != nil {
			return "", err
		}
		stringProps := make(map[string]string, len(props))
		for k, v := range props {
			stringProps[k] = fmt.Sprintf("%v", v)
		}
		return uri.AppendParameters(baseURI, stringProps), nil
	}

	return baseURI, nil
}

func getEndpointBaseURI(e v1alpha1.Endpoint) (string, error) {
	if err := validateEndpoint(e); err != nil {
		return "", err
	}

	// return the URI if explicitly stated
	if e.URI != nil {
		return *e.URI, nil
	}

	// Kamelets are a known type
	if e.Ref.Kind == v1alpha1.KameletKind {
		return fmt.Sprintf("kamelet:%s", url.PathEscape(e.Ref.Name)), nil
	}

	// assume we're using Knative for the time being (Kafka resources may be added in the future)
	return uri.AppendParameters(fmt.Sprintf("knative:endpoint/%s", url.PathEscape(e.Ref.Name)), map[string]string{
		"apiVersion": e.Ref.APIVersion,
		"kind":       e.Ref.Kind,
	}), nil
}

func validateEndpoint(e v1alpha1.Endpoint) error {
	if e.Ref == nil && e.URI == nil {
		return errors.New("no ref or URI specified in endpoint")
	} else if e.Ref != nil && e.URI != nil {
		return errors.New("cannot use both ref and URI to specify an endpoint: only one of them should be used")
	}
	return nil
}
