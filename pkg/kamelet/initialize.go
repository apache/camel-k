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

package kamelet

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
)

func Initialize(kamelet *v1alpha1.Kamelet) (*v1alpha1.Kamelet, error) {
	target := kamelet.DeepCopy()

	ok := true
	if !v1alpha1.ValidKameletName(kamelet.Name) {
		ok = false
		target.Status.SetCondition(
			v1alpha1.KameletConditionReady,
			corev1.ConditionFalse,
			v1alpha1.KameletConditionReasonInvalidName,
			fmt.Sprintf("Kamelet name %q is reserved", kamelet.Name),
		)
	}
	if !v1alpha1.ValidKameletProperties(kamelet) {
		ok = false
		target.Status.SetCondition(
			v1alpha1.KameletConditionReady,
			corev1.ConditionFalse,
			v1alpha1.KameletConditionReasonInvalidProperty,
			fmt.Sprintf("Kamelet property %q is reserved and cannot be part of the schema", v1alpha1.KameletIDProperty),
		)
	}

	if !ok {
		target.Status.Phase = v1alpha1.KameletPhaseError
	} else {
		target.Status.Phase = v1alpha1.KameletPhaseReady
		target.Status.SetCondition(
			v1alpha1.KameletConditionReady,
			corev1.ConditionTrue,
			"",
			"",
		)
		if err := recomputeProperties(target); err != nil {
			return nil, err
		}
	}

	return target, nil
}

func recomputeProperties(kamelet *v1alpha1.Kamelet) error {
	if kamelet.Spec.Definition == nil {
		return nil
	}

	kamelet.Status.Properties = make([]v1alpha1.KameletProperty, 0, len(kamelet.Spec.Definition.Properties))
	propSet := make(map[string]bool)
	for k, v := range kamelet.Spec.Definition.Properties {
		if propSet[k] {
			continue
		}
		propSet[k] = true
		defValue := ""
		if v.Default != nil {
			var val interface{}
			d := json.NewDecoder(bytes.NewReader(v.Default.RawMessage))
			d.UseNumber()
			if err := d.Decode(&val); err != nil {
				return errors.Wrapf(err, "cannot decode default value for property %q", k)
			}
			defValue = fmt.Sprintf("%v", val)
		}
		kamelet.Status.Properties = append(kamelet.Status.Properties, v1alpha1.KameletProperty{
			Name:    k,
			Default: defValue,
		})
	}
	sort.SliceStable(kamelet.Status.Properties, func(i, j int) bool {
		pi := kamelet.Status.Properties[i].Name
		pj := kamelet.Status.Properties[j].Name
		return pi < pj
	})
	return nil
}
