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

package patch

import (
	"reflect"

	jsonpatch "github.com/evanphx/json-patch"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/util/json"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

func PositiveMergePatch(source runtime.Object, target runtime.Object) ([]byte, error) {
	sourceJSON, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}

	targetJSON, err := json.Marshal(target)
	if err != nil {
		return nil, err
	}

	mergePatch, err := jsonpatch.CreateMergePatch(sourceJSON, targetJSON)
	if err != nil {
		return nil, err
	}

	var positivePatch map[string]interface{}
	err = json.Unmarshal(mergePatch, &positivePatch)
	if err != nil {
		return nil, err
	}

	// The following is a work-around to remove null fields from the JSON merge patch,
	// so that values defaulted by controllers server-side are not deleted.
	// It's generally acceptable as these values are orthogonal to the values managed
	// by the traits.
	removeNilValues(reflect.ValueOf(positivePatch), reflect.Value{})

	// Return an empty patch if no keys remain
	if len(positivePatch) == 0 {
		return make([]byte, 0), nil
	}

	return json.Marshal(positivePatch)
}

func PositiveApplyPatch(source runtime.Object) (ctrl.Object, error) {
	sourceJSON, err := json.Marshal(source)
	if err != nil {
		return nil, err
	}

	var positivePatch map[string]interface{}
	err = json.Unmarshal(sourceJSON, &positivePatch)
	if err != nil {
		return nil, err
	}

	// The following is a work-around to remove null fields from the apply patch,
	// so that ownership is not taken for non-managed fields.
	// See https://github.com/kubernetes/enhancements/tree/master/keps/sig-api-machinery/2155-clientgo-apply
	removeNilValues(reflect.ValueOf(positivePatch), reflect.Value{})

	return &unstructured.Unstructured{Object: positivePatch}, nil
}

func removeNilValues(v reflect.Value, parent reflect.Value) {
	for v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		v = v.Elem()
	}
	switch v.Kind() {
	case reflect.Array, reflect.Slice:
		for i := 0; i < v.Len(); i++ {
			removeNilValues(v.Index(i), v)
		}
	case reflect.Map:
		for _, k := range v.MapKeys() {
			switch c := v.MapIndex(k); {
			case !c.IsValid():
				// Skip keys previously deleted
				continue
			case c.IsNil(), c.Elem().Kind() == reflect.Map && len(c.Elem().MapKeys()) == 0:
				v.SetMapIndex(k, reflect.Value{})
			default:
				removeNilValues(c, v)
			}
		}
		// Back process the parent map in case it has been emptied so that it's deleted as well
		if len(v.MapKeys()) == 0 && parent.Kind() == reflect.Map {
			removeNilValues(parent, reflect.Value{})
		}
	}
}
