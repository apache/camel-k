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

package source

import (
	"fmt"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"

	corev1 "k8s.io/api/core/v1"
)

// Resolve --
func Resolve(sources []v1alpha1.SourceSpec, mapLookup func(string) (*corev1.ConfigMap, error)) ([]v1alpha1.SourceSpec, error) {
	for i := 0; i < len(sources); i++ {
		// copy the source to avoid modifications to the
		// original source
		s := sources[i].DeepCopy()

		// if it is a reference, get the content from the
		// referenced ConfigMap
		if s.ContentRef != "" {
			//look up the ConfigMap from the kubernetes cluster
			cm, err := mapLookup(s.ContentRef)
			if err != nil {
				return []v1alpha1.SourceSpec{}, err
			}

			if cm == nil {
				return []v1alpha1.SourceSpec{}, fmt.Errorf("unable to find a ConfigMap with name: %s ", s.ContentRef)
			}

			//
			// Replace ref source content with real content
			//
			s.Content = cm.Data["content"]
			s.ContentRef = ""
		}

		sources[i] = *s
	}

	return sources, nil
}
