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

package v1alpha1

import (
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// **********************************
//
// Methods
//
// **********************************

func (spec PropertySpec) String() string {
	return fmt.Sprint("%s=%s", spec.Name, spec.Value)
}

func (spec EnvironmentSpec) String() string {
	return fmt.Sprint("%s=%s", spec.Name, spec.Value)
}

// **********************************
//
// Helpers
//
// **********************************

func NewIntegrationContext(namespace string, name string) IntegrationContext {
	return IntegrationContext{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationContextKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: namespace,
			Name:      name,
		},
	}
}

func NewIntegrationContextList() IntegrationContextList {
	return IntegrationContextList{
		TypeMeta: metav1.TypeMeta{
			APIVersion: SchemeGroupVersion.String(),
			Kind:       IntegrationContextKind,
		},
	}
}
