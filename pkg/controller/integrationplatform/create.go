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

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/install"
	p "github.com/apache/camel-k/pkg/platform"
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

func (action *createAction) CanHandle(platform *v1alpha1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseCreating
}

func (action *createAction) Handle(ctx context.Context, platform *v1alpha1.IntegrationPlatform) error {
	for k := range deploy.Resources {
		if strings.HasPrefix(k, "camel-catalog-") {
			action.L.Infof("Installing camel catalog: %s", k)
			err := install.Resources(ctx, action.client, platform.Namespace, k)
			if err != nil {
				return err
			}
		}
	}

	if l := len(platform.Spec.Resources.Contexts); l > 0 {
		res := make([]string, 0, l)

		for _, c := range platform.Spec.Resources.Contexts {
			if c == p.NoContext {
				// Signals nothing to install
				continue
			}

			//
			// Assuming that if the resource ends with a yaml extension, the full
			// resource name is provided
			//
			if !strings.HasSuffix(c, ".yaml") && !strings.HasSuffix(c, ".yml") {
				c = fmt.Sprintf("platform-integration-context-%s.yaml", c)
			}

			res = append(res, c)
		}

		if len(res) > 0 {
			action.L.Info("Installing custom platform resources")
			err := install.Resources(ctx, action.client, platform.Namespace, res...)
			if err != nil {
				return err
			}
		}
	} else {
		action.L.Info("Installing default platform resources")
		err := install.Resources(ctx, action.client, platform.Namespace, p.DefaultContexts...)
		if err != nil {
			return err
		}

		if platform.Spec.Profile == v1alpha1.TraitProfileKnative {
			action.L.Info("Installing knative resources")
			err := install.Resources(ctx, action.client, platform.Namespace, p.KnativeContexts...)
			if err != nil {
				return err
			}
		}
	}

	target := platform.DeepCopy()
	target.Status.Phase = v1alpha1.IntegrationPlatformPhaseStarting
	action.L.Info("IntegrationPlatform state transition", "phase", target.Status.Phase)

	return action.client.Update(ctx, target)
}
