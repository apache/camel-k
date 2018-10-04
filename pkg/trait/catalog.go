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
	tExpose = &exposeTrait{}
	tBase = &baseTrait{}
	tOwner = &ownerTrait{}
)

// CustomizersFor returns a Catalog for the given integration details
func CustomizersFor(environment Environment) Customizer {
	switch environment.Platform.Spec.Cluster {
	case v1alpha1.IntegrationPlatformClusterOpenShift:
		return compose(
			tBase,
			tExpose,
			tOwner,
		)
	case v1alpha1.IntegrationPlatformClusterKubernetes:
		return compose(
			tBase,
			tExpose,
			tOwner,
		)
		// case Knative: ...
	}
	return nil
}

func compose(traits ...Customizer) Customizer {
	if len(traits) == 0 {
		return &identityTrait{}
	} else if len(traits) == 1 {
		return traits[0]
	}
	var composite Customizer = &identityTrait{}
	for _, t := range traits {
		composite = &chainedCustomizer{
			t1: composite,
			t2: t,
		}
	}
	return composite
}

// -------------------------------------------

type chainedCustomizer struct {
	t1 Customizer
	t2 Customizer
}

func (c *chainedCustomizer) ID() ID {
	return ID("")
}

func (c *chainedCustomizer) Customize(environment Environment, resources *kubernetes.Collection) (bool, error) {
	atLeastOnce := false
	var done bool
	var err error
	if done, err = c.t1.Customize(environment, resources); err != nil {
		return false, err
	} else if done && c.t1.ID() != "" {
		environment.ExecutedCustomizers = append(environment.ExecutedCustomizers, c.t1.ID())
	}
	atLeastOnce = atLeastOnce || done
	done2, err := c.t2.Customize(environment, resources)
	if done2 && c.t2.ID() != "" {
		environment.ExecutedCustomizers = append(environment.ExecutedCustomizers, c.t2.ID())
	}
	return atLeastOnce || done2, err
}
