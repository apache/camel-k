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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"gopkg.in/yaml.v2"
)

// YAMLFlowInspector --
type YAMLFlowInspector struct {
	baseInspector
}

// Extract --
func (i YAMLFlowInspector) Extract(source v1alpha1.SourceSpec, meta *Metadata) error {
	var flows []v1alpha1.Flow

	if err := yaml.Unmarshal([]byte(source.Content), &flows); err != nil {
		return nil
	}

	for _, flow := range flows {
		if flow.Steps[0].URI != "" {
			meta.FromURIs = append(meta.FromURIs, flow.Steps[0].URI)
		}

		for i := 1; i < len(flow.Steps); i++ {
			if flow.Steps[i].URI != "" {
				meta.ToURIs = append(meta.ToURIs, flow.Steps[i].URI)
			}
		}
	}

	meta.Dependencies = i.discoverDependencies(source, meta)

	return nil
}
