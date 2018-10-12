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
	"fmt"
	"reflect"

	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
)

// Identifiable represent an identifiable type
type Identifiable interface {
	ID() ID
}

// ID uniquely identifies a trait
type ID string

// Trait --
type Trait struct {
	Identifiable

	id      ID
	Enabled bool `property:"enabled"`
}

// ID returns the trait ID
func (trait *Trait) ID() ID {
	return trait.id
}

// NewTrait creates a new trait with defaults
func NewTrait() Trait {
	return Trait{
		Enabled: true,
	}
}

// NewTraitWithID creates a new trait with defaults and given ID
func NewTraitWithID(traitID ID) Trait {
	return Trait{
		id:      traitID,
		Enabled: true,
	}
}

// A Customizer performs customization of the deployed objects
type customizer interface {
	Identifiable
	// Customize executes the trait customization on the resources and return true if the resources have been changed
	customize(environment *environment, resources *kubernetes.Collection) (bool, error)
}

// A environment provides the context where the trait is executed
type environment struct {
	Platform            *v1alpha1.IntegrationPlatform
	Context             *v1alpha1.IntegrationContext
	Integration         *v1alpha1.Integration
	ExecutedCustomizers []ID
}

func (e *environment) getTrait(traitID ID, target interface{}) (bool, error) {
	if spec := e.getTraitSpec(traitID); spec != nil {
		err := spec.Decode(&target)
		if err != nil {
			return false, errors.Wrap(err, fmt.Sprintf("unable to convert trait %s to the target struct %s", traitID, reflect.TypeOf(target).Name()))
		}

		return true, nil
	}

	return false, nil
}

func (e *environment) getTraitSpec(traitID ID) *v1alpha1.IntegrationTraitSpec {
	if e.Integration.Spec.Traits == nil {
		return nil
	}
	if conf, ok := e.Integration.Spec.Traits[string(traitID)]; ok {
		return &conf
	}
	return nil
}

func (e *environment) isEnabled(traitID ID) bool {
	t := NewTrait()
	if _, err := e.getTrait(traitID, &t); err != nil {
		logrus.Panic(err)
	}

	return t.Enabled
}

func (e *environment) isAutoDetectionMode(traitID ID) bool {
	spec := e.getTraitSpec(traitID)
	if spec == nil {
		return true
	}

	if spec.Configuration == nil {
		return true
	}

	return spec.Configuration["enabled"] == ""
}
