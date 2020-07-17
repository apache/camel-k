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
	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/openshift"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
)

type platformTrait struct {
	BaseTrait
	v1.PlatformTrait
}

func newPlatformTrait() Trait {
	return &platformTrait{
		BaseTrait: NewBaseTrait("platform", 100),
	}
}

func (t *platformTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseNone, v1.IntegrationPhaseWaitingForPlatform) {
		return false, nil
	}

	if t.Auto == nil || !*t.Auto {
		if e.Platform == nil && t.CreateDefault == nil {
			// Calculate if the platform should be automatically created when missing.
			if ocp, err := openshift.IsOpenShift(t.Client); err != nil {
				return false, err
			} else if ocp {
				t.CreateDefault = &ocp
			}
		}
	}

	return true, nil
}

func (t *platformTrait) Apply(e *Environment) error {
	initial := e.Integration.DeepCopy()

	pl, err := t.getOrCreatePlatform(e)
	if err != nil || pl.Status.Phase != v1.IntegrationPlatformPhaseReady {
		e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForPlatform
	} else {
		e.Integration.Status.Phase = v1.IntegrationPhaseInitialization
	}

	if initial.Status.Phase != e.Integration.Status.Phase {
		if err != nil {
			e.Integration.Status.SetErrorCondition(v1.IntegrationConditionPlatformAvailable, v1.IntegrationConditionPlatformAvailableReason, err)
		}

		if pl != nil {
			e.Integration.SetIntegrationPlatform(pl)
		}
	}

	return nil
}

func (t *platformTrait) getOrCreatePlatform(e *Environment) (*v1.IntegrationPlatform, error) {
	pl, err := platform.GetOrLookupAny(t.Ctx, t.Client, e.Integration.Namespace, e.Integration.Status.Platform)
	if err != nil && k8serrors.IsNotFound(err) {
		if t.CreateDefault != nil && *t.CreateDefault {
			platformName := e.Integration.Status.Platform
			if platformName == "" {
				platformName = platform.DefaultPlatformName
			}
			defaultPlatform := v1.NewIntegrationPlatform(e.Integration.Namespace, platformName)
			if defaultPlatform.Labels == nil {
				defaultPlatform.Labels = make(map[string]string)
			}
			defaultPlatform.Labels["camel.apache.org/platform.generated"] = True
			pl = &defaultPlatform
			e.Resources.Add(pl)
			return pl, nil
		}
	}
	return pl, err
}

// IsPlatformTrait overrides base class method
func (t *platformTrait) IsPlatformTrait() bool {
	return true
}

// RequiresIntegrationPlatform overrides base class method
func (t *platformTrait) RequiresIntegrationPlatform() bool {
	return false
}
