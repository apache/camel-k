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

package integrationplatform

import (
	"context"
	"fmt"
	"strings"

	"github.com/apache/camel-k/deploy"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/install"
)

// NewCreateAction returns a action that creates resources needed by the platform
func NewCreateAction() Action {
	return &createAction{}
}

type createAction struct {
	baseAction
}

func (action *createAction) Name() string {
	return "create"
}

func (action *createAction) CanHandle(platform *v1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1.IntegrationPlatformPhaseCreating
}

func (action *createAction) Handle(ctx context.Context, platform *v1.IntegrationPlatform) (*v1.IntegrationPlatform, error) {
	for _, k := range deploy.Resources("/") {
		if strings.HasPrefix(k, "camel-catalog-") {
			action.L.Infof("Installing camel catalog: %s", k)
			err := install.Resources(ctx, action.client, platform.Namespace, install.IdentityResourceCustomizer, k)
			if err != nil {
				return nil, err
			}
		}
	}

	if l := len(platform.Status.Resources.Kits); l > 0 {
		res := make([]string, 0, l)

		for _, c := range platform.Status.Resources.Kits {
			//
			// Assuming that if the resource ends with a yaml extension, the full
			// resource name is provided
			//
			if !strings.HasSuffix(c, ".yaml") && !strings.HasSuffix(c, ".yml") {
				c = fmt.Sprintf("platform-integration-kit-%s.yaml", c)
			}

			res = append(res, c)
		}

		if len(res) > 0 {
			action.L.Info("Installing custom platform resources")
			err := install.Resources(ctx, action.client, platform.Namespace, install.IdentityResourceCustomizer, res...)
			if err != nil {
				return nil, err
			}
		}
	}

	platform.Status.Phase = v1.IntegrationPlatformPhaseReady

	return platform, nil
}
