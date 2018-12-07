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
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
)

// True --
const True = "true"

// Apply --
func Apply(integration *v1alpha1.Integration, ctx *v1alpha1.IntegrationContext) (*Environment, error) {
	environment, err := newEnvironment(integration, ctx)
	if err != nil {
		return nil, err
	}

	catalog := NewCatalog()

	// invoke the trait framework to determine the needed resources
	if err := catalog.apply(environment); err != nil {
		return nil, errors.Wrap(err, "error during trait customization before deployment")
	}

	return environment, nil
}

// newEnvironment creates a Environment from the given data
func newEnvironment(integration *v1alpha1.Integration, ctx *v1alpha1.IntegrationContext) (*Environment, error) {
	if integration == nil && ctx == nil {
		return nil, errors.New("neither integration nor context are ste")
	}

	namespace := ""
	if integration != nil {
		namespace = integration.Namespace
	} else if ctx != nil {
		namespace = ctx.Namespace
	}

	pl, err := platform.GetCurrentPlatform(namespace)
	if err != nil {
		return nil, err
	}

	if ctx == nil {
		ctx, err = GetIntegrationContext(integration)
		if err != nil {
			return nil, err
		}
	}

	return &Environment{
		Platform:       pl,
		Context:        ctx,
		Integration:    integration,
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		EnvVars:        make(map[string]string),
	}, nil
}
