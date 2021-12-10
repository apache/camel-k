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

package keda

import (
	"github.com/apache/camel-k/pkg/trait"
)

// The Keda trait can be used for automatic integration with Keda autoscalers.
//
// The Keda trait is disabled by default.
//
// +camel-k:trait=keda.
type kedaTrait struct {
	trait.BaseTrait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Metadata
	Metadata map[string]string `property:"metadata" json:"metadata,omitempty"`
}

// NewKedaTrait --.
func NewKedaTrait() trait.Trait {
	return &kedaTrait{
		BaseTrait: trait.NewBaseTrait("keda", trait.TraitOrderPostProcessResources),
	}
}

func (t *kedaTrait) Configure(e *trait.Environment) (bool, error) {

	return false, nil
}

func (t *kedaTrait) Apply(e *trait.Environment) error {
	return nil
}
