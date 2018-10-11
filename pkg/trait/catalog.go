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

var (
	tBase    = newBaseTrait()
	tService = newServiceTrait()
	tRoute   = newRouteTrait()
	tOwner   = newOwnerTrait()

	// UserFacing is list of user facing services traits
	UserFacing = []Identifiable{
		&tService,
	}
)

// customizersFor returns a Catalog for the given integration details
func customizersFor(environment *environment) customizer {
	switch environment.Platform.Spec.Cluster {
	case v1alpha1.IntegrationPlatformClusterOpenShift:
		return compose(
			&tBase,
			&tService,
			&tRoute,
			&tOwner,
		)
	case v1alpha1.IntegrationPlatformClusterKubernetes:
		return compose(
			&tBase,
			&tService,
			&tOwner,
		)
		// case Knative: ...
	}
	return nil
}

func compose(traits ...customizer) customizer {
	return &chainedCustomizer{
		customizers: traits,
	}
}

// -------------------------------------------

type chainedCustomizer struct {
	customizers []customizer
}

func (c *chainedCustomizer) ID() ID {
	return ID("")
}

func (c *chainedCustomizer) customize(environment *environment, resources *kubernetes.Collection) (bool, error) {
	atLeastOne := false
	for _, custom := range c.customizers {
		if environment.isEnabled(custom.ID()) || environment.isAutoDetectionMode(custom.ID()) {
			if done, err := custom.customize(environment, resources); err != nil {
				return false, err
			} else if done && custom.ID() != "" {
				environment.ExecutedCustomizers = append(environment.ExecutedCustomizers, custom.ID())
				atLeastOne = atLeastOne || done
			}
		}
	}
	return atLeastOne, nil
}
