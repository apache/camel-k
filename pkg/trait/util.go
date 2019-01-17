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

package trait

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/client"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// GetIntegrationContext retrieves the context set on the integration
func GetIntegrationContext(ctx context.Context, c client.Client, integration *v1alpha1.Integration) (*v1alpha1.IntegrationContext, error) {
	if integration.Status.Context == "" {
		return nil, nil
	}

	name := integration.Status.Context
	ictx := v1alpha1.NewIntegrationContext(integration.Namespace, name)
	key := k8sclient.ObjectKey{
		Namespace: integration.Namespace,
		Name:      name,
	}
	err := c.Get(ctx, key, &ictx)
	return &ictx, err
}

// VisitConfigurations --
func VisitConfigurations(
	configurationType string,
	context *v1alpha1.IntegrationContext,
	integration *v1alpha1.Integration,
	consumer func(string)) {

	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				consumer(c.Value)
			}
		}
	}

	if integration != nil {
		for _, c := range integration.Spec.Configuration {
			if c.Type == configurationType {
				consumer(c.Value)
			}
		}
	}
}

// VisitKeyValConfigurations --
func VisitKeyValConfigurations(
	configurationType string,
	context *v1alpha1.IntegrationContext,
	integration *v1alpha1.Integration,
	consumer func(string, string)) {

	if context != nil {
		// Add context properties first so integrations can
		// override it
		for _, c := range context.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					consumer(pair[0], pair[1])
				}
			}
		}
	}

	if integration != nil {
		for _, c := range integration.Spec.Configuration {
			if c.Type == configurationType {
				pair := strings.Split(c.Value, "=")
				if len(pair) == 2 {
					consumer(pair[0], pair[1])
				}
			}
		}
	}
}

// GetEnrichedSources returns an enriched version of the sources, with all the external content injected
func GetEnrichedSources(ctx context.Context, c client.Client, e *Environment, sources []v1alpha1.SourceSpec) ([]v1alpha1.SourceSpec, error) {
	enriched := make([]v1alpha1.SourceSpec, 0, len(sources))

	for _, s := range sources {
		content := s.Content
		if content == "" && s.ContentRef != "" {
			//
			// Try to check if the config map is among the one
			// creates for the deployment
			//
			sourceRef := s
			cm := e.Resources.GetConfigMap(func(m *corev1.ConfigMap) bool {
				return m.Name == sourceRef.ContentRef
			})

			//
			// if not, try to get it from the kubernetes cluster
			//
			if cm == nil {
				key := k8sclient.ObjectKey{
					Name:      s.ContentRef,
					Namespace: e.Integration.Namespace,
				}

				cm = &corev1.ConfigMap{
					TypeMeta: metav1.TypeMeta{
						Kind:       "ConfigMap",
						APIVersion: "v1",
					},
					ObjectMeta: metav1.ObjectMeta{
						Name:      s.ContentRef,
						Namespace: e.Integration.Namespace,
					},
				}

				if err := c.Get(ctx, key, cm); err != nil {
					return nil, err
				}
			}

			if cm == nil {
				return nil, fmt.Errorf("unable to find a ConfigMap with name: %s in the namespace: %s", s.ContentRef, e.Integration.Namespace)
			}

			content = cm.Data["content"]
		}
		newSource := s
		newSource.Content = content
		enriched = append(enriched, newSource)
	}
	return enriched, nil
}
