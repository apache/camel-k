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
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/pointer"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/openshift"
)

type platformTrait struct {
	BaseTrait
	traitv1.PlatformTrait `property:",squash"`
}

func newPlatformTrait() Trait {
	return &platformTrait{
		BaseTrait: NewBaseTrait("platform", 100),
	}
}

func (t *platformTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseNone, v1.IntegrationPhaseWaitingForPlatform) {
		return false, nil
	}

	if !pointer.BoolDeref(t.Auto, false) {
		if e.Platform == nil {
			if t.CreateDefault == nil {
				// Calculate if the platform should be automatically created when missing.
				if ocp, err := openshift.IsOpenShift(t.Client); err != nil {
					return false, err
				} else if ocp {
					t.CreateDefault = &ocp
				}
			}
			if t.Global == nil {
				globalOperator := platform.IsCurrentOperatorGlobal()
				t.Global = &globalOperator
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
	pl, err := platform.GetOrFindForResource(e.Ctx, t.Client, e.Integration, false)
	if err != nil && k8serrors.IsNotFound(err) {
		if pointer.BoolDeref(t.CreateDefault, false) {
			platformName := e.Integration.Status.Platform
			if platformName == "" {
				platformName = platform.DefaultPlatformName
			}
			namespace := e.Integration.Namespace
			if pointer.BoolDeref(t.Global, false) {
				operatorNamespace := platform.GetOperatorNamespace()
				if operatorNamespace != "" {
					namespace = operatorNamespace
				}
			}
			defaultPlatform := v1.NewIntegrationPlatform(namespace, platformName)
			if defaultPlatform.Labels == nil {
				defaultPlatform.Labels = make(map[string]string)
			}
			defaultPlatform.Labels["camel.apache.org/platform.generated"] = True
			// Cascade the operator id in charge to reconcile the Integration
			if v1.GetOperatorIDAnnotation(e.Integration) != "" {
				if defaultPlatform.Annotations == nil {
					defaultPlatform.Annotations = make(map[string]string)
				}
				defaultPlatform.SetOperatorID(v1.GetOperatorIDAnnotation(e.Integration))
			}
			pl = &defaultPlatform
			e.Resources.Add(pl)
			return pl, nil
		}
	}
	return pl, err
}

// IsPlatformTrait overrides base class method.
func (t *platformTrait) IsPlatformTrait() bool {
	return true
}

// RequiresIntegrationPlatform overrides base class method.
func (t *platformTrait) RequiresIntegrationPlatform() bool {
	return false
}
