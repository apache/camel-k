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
	apierrors "k8s.io/apimachinery/pkg/api/errors"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/platform"
)

const (
	platformTraitID    = "platform"
	platformTraitOrder = 100
)

type platformTrait struct {
	BasePlatformTrait
	traitv1.PlatformTrait `property:",squash"`
}

func newPlatformTrait() Trait {
	return &platformTrait{
		BasePlatformTrait: NewBasePlatformTrait(platformTraitID, platformTraitOrder),
	}
}

func (t *platformTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || e.Integration.IsSynthetic() {
		// Don't run this trait for a synthetic integration
		return false, nil, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseNone, v1.IntegrationPhaseWaitingForPlatform) {
		return false, nil, nil
	}

	return true, nil, nil
}

func (t *platformTrait) Apply(e *Environment) error {
	initial := e.Integration.DeepCopy()

	pl, err := t.getOrCreatePlatform(e)
	// Do not change to Initialization phase within the trait
	switch {
	case err != nil:
		e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForPlatform
		if initial.Status.Phase != e.Integration.Status.Phase {
			e.Integration.Status.SetErrorCondition(
				v1.IntegrationConditionPlatformAvailable,
				v1.IntegrationConditionPlatformAvailableReason,
				err)

			if pl != nil {
				e.Integration.SetIntegrationPlatform(pl)
			}
		}
	case pl == nil:
		e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForPlatform
	case pl.Status.Phase != v1.IntegrationPlatformPhaseReady:
		e.Integration.Status.Phase = v1.IntegrationPhaseWaitingForPlatform
		if initial.Status.Phase != e.Integration.Status.Phase {
			e.Integration.SetIntegrationPlatform(pl)
		}
	default:
		// In success case, phase should be reset to none
		e.Integration.Status.Phase = v1.IntegrationPhaseNone
		e.Integration.SetIntegrationPlatform(pl)
	}

	return nil
}

func (t *platformTrait) getOrCreatePlatform(e *Environment) (*v1.IntegrationPlatform, error) {
	pl, err := platform.GetForResource(e.Ctx, t.Client, e.Integration)
	if err != nil && !apierrors.IsNotFound(err) {
		return nil, err
	}

	return pl, nil
}

// RequiresIntegrationPlatform overrides base class method.
func (t *platformTrait) RequiresIntegrationPlatform() bool {
	return false
}
