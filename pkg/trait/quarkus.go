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
	"github.com/apache/camel-k/pkg/builder"
)

// The Quarkus trait activates the Quarkus runtime.
//
// It's enabled by default.
//
// +camel-k:trait=quarkus
type quarkusTrait struct {
	BaseTrait `property:",squash"`
	// The Quarkus runtime type (reserved for future use)
	Native bool `property:"native" json:"native,omitempty"`
}

func newQuarkusTrait() Trait {
	return &quarkusTrait{
		BaseTrait: NewBaseTrait("quarkus", 700),
	}
}

func (t *quarkusTrait) Configure(e *Environment) (bool, error) {
	return IsNilOrTrue(t.Enabled), nil
}

func (t *quarkusTrait) Apply(e *Environment) error {
	return nil
}

// IsPlatformTrait overrides base class method
func (t *quarkusTrait) IsPlatformTrait() bool {
	return true
}

// InfluencesKit overrides base class method
func (t *quarkusTrait) InfluencesKit() bool {
	return true
}

func (t *quarkusTrait) addBuildSteps(steps *[]builder.Step) {
	*steps = append(*steps, builder.QuarkusSteps...)
}
