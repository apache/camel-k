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

// Package bindings provides APIs to transform Kubernetes objects into Camel URIs equivalents
package bindings

import "fmt"

// AsYamlDSL construct proper Camel Yaml DSL from given binding.
func (b Binding) AsYamlDSL() map[string]interface{} {
	step := b.Step
	if step == nil {
		step = map[string]interface{}{
			"to": b.URI,
		}
	}

	return step
}

// GenerateID generates an identifier based on the context type and its optional position.
func (c EndpointContext) GenerateID() string {
	id := string(c.Type)
	if c.Position != nil {
		id = fmt.Sprintf("%s-%d", id, *c.Position)
	}
	return id
}

// GenerateID generates an identifier based on the context type and its optional position.
// Deprecated.
func (c V1alpha1EndpointContext) GenerateID() string {
	id := string(c.Type)
	if c.Position != nil {
		id = fmt.Sprintf("%s-%d", id, *c.Position)
	}
	return id
}
