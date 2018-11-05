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
	"k8s.io/apimachinery/pkg/runtime"
)

// BeforeDeployment generates all required resources for deploying the given integration
func BeforeDeployment(integration *v1alpha1.Integration) ([]runtime.Object, error) {
	environment, err := newEnvironment(integration)
	if err != nil {
		return nil, err
	}
	resources := kubernetes.NewCollection()
	catalog := NewCatalog()
	// invoke the trait framework to determine the needed resources
	if err := catalog.executeBeforeDeployment(environment, resources); err != nil {
		return nil, errors.Wrap(err, "error during trait customization before deployment")
	}
	return resources.Items(), nil
}

// BeforeInit executes custom initializazion of the integration
func BeforeInit(integration *v1alpha1.Integration) error {
	environment, err := newEnvironment(integration)
	if err != nil {
		return err
	}
	catalog := NewCatalog()
	// invoke the trait framework to determine the needed resources
	if err := catalog.executeBeforeInit(environment, integration); err != nil {
		return errors.Wrap(err, "error during trait customization before init")
	}
	return nil
}

// newEnvironment creates a environment from the given data
func newEnvironment(integration *v1alpha1.Integration) (*environment, error) {
	pl, err := platform.GetCurrentPlatform(integration.Namespace)
	if err != nil {
		return nil, err
	}
	ctx, err := GetIntegrationContext(integration)
	if err != nil {
		return nil, err
	}

	return &environment{
		Platform:       pl,
		Context:        ctx,
		Integration:    integration,
		ExecutedTraits: make([]ID, 0),
	}, nil
}
