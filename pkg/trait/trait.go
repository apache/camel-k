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

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/client"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/kubernetes"
)

func Apply(ctx context.Context, c client.Client, integration *v1.Integration, kit *v1.IntegrationKit) (*Environment, error) {
	environment, err := newEnvironment(ctx, c, integration, kit)
	if err != nil {
		return nil, err
	}

	catalog := NewCatalog(ctx, c)

	// set the catalog
	environment.Catalog = catalog

	// invoke the trait framework to determine the needed resources
	if err := catalog.apply(environment); err != nil {
		return nil, errors.Wrap(err, "error during trait customization")
	}

	// execute post actions registered by traits
	for _, postAction := range environment.PostActions {
		err := postAction(environment)
		if err != nil {
			return nil, errors.Wrap(err, "error executing post actions")
		}
	}

	return environment, nil
}

// newEnvironment creates a Environment from the given data
func newEnvironment(ctx context.Context, c client.Client, integration *v1.Integration, kit *v1.IntegrationKit) (*Environment, error) {
	if integration == nil && ctx == nil {
		return nil, errors.New("neither integration nor kit are set")
	}

	namespace := ""
	if integration != nil {
		namespace = integration.Namespace
	} else if kit != nil {
		namespace = kit.Namespace
	}

	pl, err := platform.GetCurrent(ctx, c, namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if kit == nil {
		kit, err = getIntegrationKit(ctx, c, integration)
		if err != nil {
			return nil, err
		}
	}

	env := Environment{
		C:                     ctx,
		Platform:              pl,
		Client:                c,
		IntegrationKit:        kit,
		Integration:           integration,
		ExecutedTraits:        make([]Trait, 0),
		Resources:             kubernetes.NewCollection(),
		EnvVars:               make([]corev1.EnvVar, 0),
		ApplicationProperties: make(map[string]string),
	}

	return &env, nil
}
