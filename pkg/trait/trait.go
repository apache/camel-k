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
	"errors"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/client"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
	"github.com/apache/camel-k/v2/pkg/util/log"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

func Apply(ctx context.Context, c client.Client, integration *v1.Integration, kit *v1.IntegrationKit) (*Environment, error) {
	var ilog log.Logger
	switch {
	case integration != nil:
		ilog = log.ForIntegration(integration)
	case kit != nil:
		ilog = log.ForIntegrationKit(kit)
	default:
		ilog = log.WithValues("Function", "trait.Apply")
	}

	environment, err := newEnvironment(ctx, c, integration, kit)
	if err != nil {
		return nil, fmt.Errorf("error creating trait environment: %w", err)
	}

	catalog := NewCatalog(c)

	// set the catalog
	environment.Catalog = catalog

	// invoke the trait framework to determine the needed resources
	conditions, err := catalog.apply(environment)
	// Conditions contains informative message coming from the trait execution and useful to be reported into it or ik CR
	// they must be applied before returning after an error
	for _, tc := range conditions {
		switch {
		case integration != nil:
			// set an Integration condition
			integration.Status.SetCondition(tc.integrationCondition())
		case kit != nil:
			// set an IntegrationKit condition
			kit.Status.SetCondition(tc.integrationKitCondition())
		}
	}
	if err != nil {
		return nil, fmt.Errorf("error during trait customization: %w", err)
	}

	// execute post actions registered by traits
	for _, postAction := range environment.PostActions {
		err := postAction(environment)
		if err != nil {
			return nil, fmt.Errorf("error executing post actions: %w", err)
		}
	}

	switch {
	case integration != nil:
		ilog.Debug("Applied traits to Integration", "integration", integration.Name, "namespace", integration.Namespace)
	case kit != nil:
		ilog.Debug("Applied traits to Integration kit", "integration kit", kit.Name, "namespace", kit.Namespace)
	default:
		ilog.Debug("Applied traits")
	}
	return environment, nil
}

// newEnvironment creates a Environment from the given data.
func newEnvironment(ctx context.Context, c client.Client, integration *v1.Integration, kit *v1.IntegrationKit) (*Environment, error) {
	if integration == nil && kit == nil {
		return nil, errors.New("neither integration nor kit are set")
	}

	var obj k8sclient.Object
	if integration != nil {
		obj = integration
	} else if kit != nil {
		obj = kit
	}

	pl, err := platform.GetForResource(ctx, c, obj)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	if kit == nil {
		kit, err = getIntegrationKit(ctx, c, integration)
		if err != nil {
			return nil, err
		}
	}

	//
	// kit can still be nil if integration kit is yet
	// to finish building and be assigned to the integration
	//
	env := Environment{
		Ctx:                   ctx,
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
