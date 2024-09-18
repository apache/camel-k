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
	"github.com/apache/camel-k/v2/pkg/install"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/util/boolean"
	"github.com/apache/camel-k/v2/pkg/util/defaults"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
	image "github.com/apache/camel-k/v2/pkg/util/registry"
)

const (
	platformTraitID    = "platform"
	platformTraitOrder = 100
)

type platformTrait struct {
	BasePlatformTrait
	traitv1.PlatformTrait `property:",squash"`
	// Parameters to be used internally
	createDefault *bool
	global        *bool
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

	if t.CreateDefault == nil && ptr.Deref(t.Auto, false) && e.Platform == nil {
		// Calculate if the platform should be automatically created when missing.
		if ocp, err := openshift.IsOpenShift(t.Client); err != nil {
			return false, nil, err
		} else if ocp {
			t.createDefault = ptr.To(true)
		} else if addr, err := image.GetRegistryAddress(e.Ctx, t.Client); err != nil {
			return false, nil, err
		} else if addr != nil {
			t.createDefault = ptr.To(true)
		}
	}

	if t.Global == nil {
		globalOperator := platform.IsCurrentOperatorGlobal()
		t.global = &globalOperator
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
	if apierrors.IsNotFound(err) && ptr.Deref(t.getCreateDefault(), false) {
		pl = t.createDefaultPlatform(e)
		e.Resources.Add(pl)

		t.installViewerRole(e, pl)
	}

	if err != nil {
		return nil, err
	}

	return pl, nil
}

// RequiresIntegrationPlatform overrides base class method.
func (t *platformTrait) RequiresIntegrationPlatform() bool {
	return false
}

func (t *platformTrait) createDefaultPlatform(e *Environment) *v1.IntegrationPlatform {
	platformName := e.Integration.Status.Platform
	if platformName == "" {
		platformName = defaults.OperatorID()
	}

	if platformName == "" {
		platformName = platform.DefaultPlatformName
	}
	namespace := e.Integration.Namespace
	if ptr.Deref(t.getGlobal(), false) {
		operatorNamespace := platform.GetOperatorNamespace()
		if operatorNamespace != "" {
			namespace = operatorNamespace
		}
	}

	defaultPlatform := v1.NewIntegrationPlatform(namespace, platformName)
	if defaultPlatform.Labels == nil {
		defaultPlatform.Labels = make(map[string]string)
	}
	defaultPlatform.Labels["app"] = "camel-k"
	defaultPlatform.Labels["camel.apache.org/platform.generated"] = boolean.TrueString

	// Cascade the operator id in charge to reconcile the Integration
	if v1.GetOperatorIDAnnotation(e.Integration) != "" {
		defaultPlatform.SetOperatorID(v1.GetOperatorIDAnnotation(e.Integration))
	}

	return &defaultPlatform
}

func (t *platformTrait) installViewerRole(e *Environment, itp *v1.IntegrationPlatform) {
	// Make sure that IntegrationPlatform installed in operator namespace can be seen by others
	err := install.IntegrationPlatformViewerRole(e.Ctx, t.Client, itp.Namespace)
	if err != nil && !apierrors.IsAlreadyExists(err) {
		t.L.Infof("Cannot install global IntegrationPlatform viewer role in namespace '%s': skipping.", itp.Namespace)
	}
}

func (t *platformTrait) getCreateDefault() *bool {
	if t.CreateDefault == nil {
		return t.createDefault
	}

	return t.CreateDefault
}

func (t *platformTrait) getGlobal() *bool {
	if t.Global == nil {
		return t.global
	}

	return t.Global
}
