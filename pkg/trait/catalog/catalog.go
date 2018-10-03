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

package catalog

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

// TraitID uniquely identifies a trait
type TraitID string

const (
	// Expose exposes a integration to the external world
	Expose TraitID = "expose"
)

// A Catalog is just a DeploymentCustomizer that applies multiple traits
type Catalog trait.DeploymentCustomizer

// For returns a Catalog for the given integration details
func For(environment trait.Environment) Catalog {

}

func compose(traits ...trait.DeploymentCustomizer) trait.DeploymentCustomizer {
	if len(traits) == 0 {
		return &identityTrait{}
	} else if len(traits) == 1 {
		return traits[0]
	}
	var composite trait.DeploymentCustomizer = &identityTrait{}
	for _, t := range traits {
		composite = &catalogCustomizer{
			t1: composite,
			t2: t,
		}
	}
	return composite
}

// -------------------------------------------

type catalogCustomizer struct {
	t1 trait.DeploymentCustomizer
	t2 trait.DeploymentCustomizer
}

func (c *catalogCustomizer) Name() string {
	return ""
}

func (c *catalogCustomizer) Customize(environment trait.Environment, resources *kubernetes.Collection) (bool, error) {
	atLeastOnce := false
	var done bool
	var err error
	if done, err = c.t1.Customize(environment, resources); err != nil {
		return false, err
	} else if done && c.t1.Name() != "" {
		environment.ExecutedCustomizers = append(environment.ExecutedCustomizers, c.t1.Name())
	}
	atLeastOnce = atLeastOnce || done
	done2, err := c.t2.Customize(environment, resources)
	if done2 && c.t2.Name() != "" {
		environment.ExecutedCustomizers = append(environment.ExecutedCustomizers, c.t2.Name())
	}
	environment.ExecutedCustomizers = append(environment.ExecutedCustomizers, c.t1.Name())
	return atLeastOnce || done2, err
}
