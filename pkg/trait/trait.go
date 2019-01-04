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
	"context"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/pkg/errors"
	"k8s.io/api/core/v1"
	"github.com/apache/camel-k/pkg/client"
)

// True --
const True = "true"

// Apply --
func Apply(ctx context.Context, c client.Client, integration *v1alpha1.Integration, ictx *v1alpha1.IntegrationContext) (*Environment, error) {
	environment, err := newEnvironment(ctx, c, integration, ictx)
	if err != nil {
		return nil, err
	}

	catalog := NewCatalog(ctx, c)

	// invoke the trait framework to determine the needed resources
	if err := catalog.apply(environment); err != nil {
		return nil, errors.Wrap(err, "error during trait customization before deployment")
	}

	return environment, nil
}

// newEnvironment creates a Environment from the given data
func newEnvironment(ctx context.Context, c client.Client, integration *v1alpha1.Integration, ictx *v1alpha1.IntegrationContext) (*Environment, error) {
	if integration == nil && ctx == nil {
		return nil, errors.New("neither integration nor context are ste")
	}

	namespace := ""
	if integration != nil {
		namespace = integration.Namespace
	} else if ictx != nil {
		namespace = ictx.Namespace
	}

	pl, err := platform.GetCurrentPlatform(ctx, c, namespace)
	if err != nil {
		return nil, err
	}

	if ictx == nil {
		ictx, err = GetIntegrationContext(ctx, c, integration)
		if err != nil {
			return nil, err
		}
	}

	return &Environment{
		Platform:       pl,
		Context:        ictx,
		Integration:    integration,
		ExecutedTraits: make([]Trait, 0),
		Resources:      kubernetes.NewCollection(),
		EnvVars:        make([]v1.EnvVar, 0),
	}, nil
}
