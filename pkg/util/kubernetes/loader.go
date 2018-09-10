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

package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"github.com/operator-framework/operator-sdk/pkg/util/k8sutil"
	"k8s.io/apimachinery/pkg/util/yaml"
	"k8s.io/apimachinery/pkg/runtime"
)

func LoadResourceFromYaml(data string) (runtime.Object, error) {
	role := []byte(data)
	roleJson, err := yaml.ToJSON(role)
	if err != nil {
		return nil, err
	}
	u := unstructured.Unstructured{}
	err = u.UnmarshalJSON(roleJson)
	if err != nil {
		return nil, err
	}

	return k8sutil.RuntimeObjectFromUnstructured(&u)
}
