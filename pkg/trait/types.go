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
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/builder"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// Identifiable represent an identifiable type
type Identifiable interface {
	ID() ID
}

// ID uniquely identifies a trait
type ID string

// Trait is the interface of all traits
type Trait interface {
	Identifiable
	// IsEnabled tells if the trait is enabled
	IsEnabled() bool
	// IsAuto determine if the trait should be configured automatically
	IsAuto() bool
	// appliesTo tells if the trait supports the given environment
	appliesTo(environment *Environment) bool
	// autoconfigure is called before any customization to ensure the trait is fully configured
	autoconfigure(environment *Environment) error
	// apply executes a customization of the Environment
	apply(environment *Environment) error
}

/* Base trait */

// BaseTrait is the root trait with noop implementations for hooks
type BaseTrait struct {
	id      ID
	Enabled *bool `property:"enabled"`
	Auto    *bool `property:"auto"`
}

func newBaseTrait(id string) BaseTrait {
	return BaseTrait{
		id: ID(id),
	}
}

// ID returns the identifier of the trait
func (trait *BaseTrait) ID() ID {
	return trait.id
}

// IsAuto determines if we should apply automatic configuration
func (trait *BaseTrait) IsAuto() bool {
	if trait.Auto == nil {
		return true
	}
	return *trait.Auto
}

// IsEnabled is used to determine if the trait needs to be executed
func (trait *BaseTrait) IsEnabled() bool {
	if trait.Enabled == nil {
		return true
	}
	return *trait.Enabled
}

func (trait *BaseTrait) autoconfigure(environment *Environment) error {
	return nil
}

func (trait *BaseTrait) apply(environment *Environment) error {
	return nil
}

/* Environment */

// A Environment provides the context where the trait is executed
type Environment struct {
	Platform       *v1alpha1.IntegrationPlatform
	Context        *v1alpha1.IntegrationContext
	Integration    *v1alpha1.Integration
	Resources      *kubernetes.Collection
	Steps          []builder.Step
	ExecutedTraits []ID
	EnvVars        map[string]string
}

// IntegrationInPhase --
func (e *Environment) IntegrationInPhase(phase v1alpha1.IntegrationPhase) bool {
	return e.Integration != nil && e.Integration.Status.Phase == phase
}

// IntegrationContextInPhase --
func (e *Environment) IntegrationContextInPhase(phase v1alpha1.IntegrationContextPhase) bool {
	return e.Context != nil && e.Context.Status.Phase == phase
}

// DeterimeProfile determines the TraitProfile of the environment.
// First looking at the Integration.Spec for a Profile,
// next looking at the Context.Spec
// and lastly the Platform Profile
func (e *Environment) DetermineProfile() v1alpha1.TraitProfile {
	if e.Integration != nil && e.Integration.Spec.Profile != "" {
		return e.Integration.Spec.Profile
	}

	if e.Context != nil && e.Context.Spec.Profile != "" {
		return e.Context.Spec.Profile
	}

	return platform.GetProfile(e.Platform)
}
