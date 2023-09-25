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
	"context"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	platformutil "github.com/apache/camel-k/v2/pkg/platform"
)

// NewInitializeAction returns a action that initializes the catalog configuration when not provided by the user.
func NewInitializeAction() Action {
	return &initializeAction{}
}

type initializeAction struct {
	baseAction
}

func (action *initializeAction) Name() string {
	return "initialize"
}

func (action *initializeAction) CanHandle(catalog *v1.CamelCatalog) bool {
	return catalog.Status.Phase == v1.CamelCatalogPhaseNone
}

func (action *initializeAction) Handle(ctx context.Context, catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	action.L.Info("Initializing CamelCatalog")

	platform, err := platformutil.GetOrFindLocal(ctx, action.client, catalog.Namespace)

	if err != nil {
		return catalog, err
	}

	if platform.Status.Phase != v1.IntegrationPlatformPhaseReady {
		// Wait the platform to be ready
		return catalog, nil
	}

	return initialize(catalog)

}

func initialize(catalog *v1.CamelCatalog) (*v1.CamelCatalog, error) {
	target := catalog.DeepCopy()
	// TODO - we may verify the existence of the catalog image (required by native build)
	// or any other condition that may make a CamelCatalog to fail.
	target.Status.Phase = v1.CamelCatalogPhaseReady

	return target, nil
}
