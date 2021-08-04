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
	"strings"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

const (
	CamelCreatorLabelPrefix = "camel.apache.org/created.by"

	CamelCreatorLabelKind      = CamelCreatorLabelPrefix + ".kind"
	CamelCreatorLabelName      = CamelCreatorLabelPrefix + ".name"
	CamelCreatorLabelNamespace = CamelCreatorLabelPrefix + ".namespace"
	CamelCreatorLabelVersion   = CamelCreatorLabelPrefix + ".version"
)

// FilterCamelCreatorLabels is used to inherit the creator information among resources
func FilterCamelCreatorLabels(source map[string]string) map[string]string {
	res := make(map[string]string)
	for k, v := range source {
		if strings.HasPrefix(k, CamelCreatorLabelPrefix) {
			res[k] = v
		}
	}
	return res
}

// MergeCamelCreatorLabels is used to inject the creator information from another set of labels
func MergeCamelCreatorLabels(source map[string]string, target map[string]string) map[string]string {
	if target == nil {
		target = make(map[string]string)
	}
	for k, v := range FilterCamelCreatorLabels(source) {
		target[k] = v
	}
	return target
}

// GetCamelCreator returns the Camel creator object referenced by this runtime object, if present
func GetCamelCreator(obj runtime.Object) *corev1.ObjectReference {
	if m, ok := obj.(metav1.Object); ok {
		kind := m.GetLabels()[CamelCreatorLabelKind]
		name := m.GetLabels()[CamelCreatorLabelName]
		namespace, ok := m.GetLabels()[CamelCreatorLabelNamespace]
		if !ok {
			namespace = m.GetNamespace()
		}
		if kind != "" && name != "" {
			return &corev1.ObjectReference{
				Kind:       kind,
				Namespace:  namespace,
				Name:       name,
				APIVersion: v1.SchemeGroupVersion.String(),
			}
		}
	}
	return nil
}
