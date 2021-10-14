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

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/platform"
	"github.com/apache/camel-k/pkg/util/openshift"
)

// The platform trait is a base trait that is used to assign an integration platform to an integration.
//
// In case the platform is missing, the trait is allowed to create a default platform.
// This feature is especially useful in contexts where there's no need to provide a custom configuration for the platform
// (e.g. on OpenShift the default settings work, since there's an embedded container image registry).
//
// +camel-k:trait=platform
type platformTrait struct {
	BaseTrait `property:",squash"`
	// To create a default (empty) platform when the platform is missing.
	CreateDefault *bool `property:"create-default" json:"createDefault,omitempty"`
	// Indicates if the platform should be created globally in the case of global operator (default true).
	Global *bool `property:"global" json:"global,omitempty"`
	// To automatically detect from the environment if a default platform can be created (it will be created on OpenShift only).
	Auto *bool `property:"auto" json:"auto,omitempty"`
}

func newPlatformTrait() Trait {
	return &platformTrait{
		BaseTrait: NewBaseTrait("platform", 100),
	}
}

func (t *platformTrait) Configure(e *Environment) (bool, error) {
	if IsFalse(t.Enabled) {
		return false, nil
	}

	if !e.IntegrationInPhase(v1.IntegrationPhaseNone, v1.IntegrationPhaseWaitingForPlatform) {
		return false, nil
	}

	if IsNilOrFalse(t.Auto) {
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
		if IsTrue(t.CreateDefault) {
			platformName := e.Integration.Status.Platform
			if platformName == "" {
				platformName = platform.DefaultPlatformName
			}
			namespace := e.Integration.Namespace
			if IsTrue(t.Global) {
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
