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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/bindings"
	"github.com/apache/camel-k/pkg/util/kubernetes"
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

	from, err := bindings.Translate(v1alpha1.EndpointTypeSource, kameletbinding.Spec.Source)
	if err != nil {
		return nil, errors.Wrap(err, "could not determine source URI")
	}
	to, err := bindings.Translate(v1alpha1.EndpointTypeSink, kameletbinding.Spec.Sink)
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

	target := kameletbinding.DeepCopy()
	target.Status.Phase = v1alpha1.KameletBindingPhaseCreating
	return target, nil
}
