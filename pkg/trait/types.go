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
	"k8s.io/api/core/v1"
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

	// Configure the trait
	Configure(environment *Environment) (bool, error)

	// Apply executes a customization of the Environment
	Apply(environment *Environment) error
}

/* Base trait */

// BaseTrait is the root trait with noop implementations for hooks
type BaseTrait struct {
	id      ID
	Enabled *bool `property:"enabled"`
}

// ID returns the identifier of the trait
func (trait *BaseTrait) ID() ID {
	return trait.id
}

/* Environment */

// A Environment provides the context where the trait is executed
type Environment struct {
	Platform       *v1alpha1.IntegrationPlatform
	Context        *v1alpha1.IntegrationContext
	Integration    *v1alpha1.Integration
	Resources      *kubernetes.Collection
	Steps          []builder.Step
	BuildDir       string
	ExecutedTraits []Trait
	EnvVars        []v1.EnvVar
}

// GetTrait --
func (e *Environment) GetTrait(id ID) Trait {
	for _, t := range e.ExecutedTraits {
		if t.ID() == id {
			return t
		}
	}

	return nil
}

// IntegrationInPhase --
func (e *Environment) IntegrationInPhase(phase v1alpha1.IntegrationPhase) bool {
	return e.Integration != nil && e.Integration.Status.Phase == phase
}

// IntegrationContextInPhase --
func (e *Environment) IntegrationContextInPhase(phase v1alpha1.IntegrationContextPhase) bool {
	return e.Context != nil && e.Context.Status.Phase == phase
}

// InPhase --
func (e *Environment) InPhase(c v1alpha1.IntegrationContextPhase, i v1alpha1.IntegrationPhase) bool {
	return e.IntegrationContextInPhase(c) && e.IntegrationInPhase(i)
}

// DetermineProfile determines the TraitProfile of the environment.
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
