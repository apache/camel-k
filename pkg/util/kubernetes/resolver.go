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
	"context"
	"fmt"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	corev1 "k8s.io/api/core/v1"
)

// ResolveSources --
func ResolveSources(elements []v1alpha1.SourceSpec, mapLookup func(string) (*corev1.ConfigMap, error)) ([]v1alpha1.SourceSpec, error) {
	for i := 0; i < len(elements); i++ {
		r := &elements[i]

		if err := Resolve(&r.DataSpec, mapLookup); err != nil {
			return nil, err
		}
	}

	return elements, nil
}

// ResolveResource --
func ResolveResource(elements []v1alpha1.ResourceSpec, mapLookup func(string) (*corev1.ConfigMap, error)) ([]v1alpha1.ResourceSpec, error) {
	for i := 0; i < len(elements); i++ {
		r := &elements[i]

		if err := Resolve(&r.DataSpec, mapLookup); err != nil {
			return nil, err
		}
	}

	return elements, nil
}

// Resolve --
func Resolve(data *v1alpha1.DataSpec, mapLookup func(string) (*corev1.ConfigMap, error)) error {
	// if it is a reference, get the content from the
	// referenced ConfigMap
	if data.ContentRef != "" {
		//look up the ConfigMap from the kubernetes cluster
		cm, err := mapLookup(data.ContentRef)
		if err != nil {
			return err
		}

		if cm == nil {
			return fmt.Errorf("unable to find a ConfigMap with name: %s ", data.ContentRef)
		}

		//
		// Replace ref source content with real content
		//
		data.Content = cm.Data["content"]
		data.ContentRef = ""
	}

	return nil
}

// ResolveIntegrationSources --
func ResolveIntegrationSources(context context.Context, client client.Client, integration *v1alpha1.Integration, resources *Collection) ([]v1alpha1.SourceSpec, error) {
	if integration == nil {
		return nil, nil
	}

	return ResolveSources(integration.Sources(), func(name string) (*corev1.ConfigMap, error) {
		// the config map could be part of the resources created
		// by traits
		cm := resources.GetConfigMap(func(m *corev1.ConfigMap) bool {
			return m.Name == name
		})

		if cm != nil {
			return cm, nil
		}

		return GetConfigMap(context, client, name, integration.Namespace)
	})
}

// ResolveIntegrationResources --
// nolint: lll
func ResolveIntegrationResources(context context.Context, client client.Client, integration *v1alpha1.Integration, resources *Collection) ([]v1alpha1.ResourceSpec, error) {
	if integration == nil {
		return nil, nil
	}

	return ResolveResource(integration.Spec.Resources, func(name string) (*corev1.ConfigMap, error) {
		// the config map could be part of the resources created
		// by traits
		cm := resources.GetConfigMap(func(m *corev1.ConfigMap) bool {
			return m.Name == name
		})

		if cm != nil {
			return cm, nil
		}

		return GetConfigMap(context, client, name, integration.Namespace)
	})
}
