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
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// Identifiable represent an identifiable type
type Identifiable interface {
	ID() ID
}

// ID uniquely identifies a trait
type ID string

// ITrait TODO rename
type ITrait interface {
	Identifiable
	// enabled tells if the trait is enabled
	IsEnabled() bool
	// auto determine if the trait should be configured automatically
	IsAuto() bool
	// autoconfigure is called before any customization to ensure the trait is fully configured
	autoconfigure(environment *environment, resources *kubernetes.Collection) error
	// customize executes the trait customization on the resources and return true if the resources have been changed
	customize(environment *environment, resources *kubernetes.Collection) error
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

func (trait *BaseTrait) autoconfigure(environment *environment, resources *kubernetes.Collection) error {
	return nil
}

func (trait *BaseTrait) customize(environment *environment, resources *kubernetes.Collection) error {
	return nil
}

/* Environment */

// A environment provides the context where the trait is executed
type environment struct {
	Platform       *v1alpha1.IntegrationPlatform
	Context        *v1alpha1.IntegrationContext
	Integration    *v1alpha1.Integration
	ExecutedTraits []ID
}
