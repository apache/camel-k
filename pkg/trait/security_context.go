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
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/util/openshift"
)

const (
	securityContextTraitID   = "security-context"
	securityContextTraitOder = 1600

	defaultPodRunAsNonRoot       = false
	defaultPodSeccompProfileType = corev1.SeccompProfileTypeRuntimeDefault
)

type securityContextTrait struct {
	BasePlatformTrait
	traitv1.SecurityContextTrait `property:",squash"`
}

func newSecurityContextTrait() Trait {
	return &securityContextTrait{
		BasePlatformTrait: NewBasePlatformTrait(securityContextTraitID, securityContextTraitOder),
	}
}

func (t *securityContextTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil {
		return false, nil, nil
	}
	if !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	// We better disable because "PodSecurityContext properties can affect non-user sidecar containers that come from Knative or your service mesh".
	// https://knative.dev/docs/serving/configuration/feature-flags/#kubernetes-security-context
	// The user should instead use container security context.
	condition := e.Integration.Status.GetCondition(v1.IntegrationConditionKnativeServiceAvailable)
	if condition != nil && condition.Status == corev1.ConditionTrue {
		return false, NewIntegrationConditionPlatformDisabledWithMessage(
			"SecurityContext",
			"pod security context is disabled for Knative Service. "+
				"PodSecurityContext properties can affect non-user sidecar containers that come from Knative or your service mesh. "+
				"Use container security context instead.",
		), nil
	}

	return true, nil, nil
}

func (t *securityContextTrait) Apply(e *Environment) error {
	podSpec := e.GetIntegrationPodSpec()
	if podSpec == nil {
		return fmt.Errorf("could not find any integration deployment for %v", e.Integration.Name)
	}

	return t.setSecurityContext(e, podSpec)
}

func (t *securityContextTrait) setSecurityContext(e *Environment, podSpec *corev1.PodSpec) error {
	sc := corev1.PodSecurityContext{
		RunAsNonRoot: t.getRunAsNonRoot(),
		SeccompProfile: &corev1.SeccompProfile{
			Type: t.getSeccompProfileType(),
		},
	}

	runAsUser, err := t.getUser(e)
	if err != nil {
		return err
	}

	t.RunAsUser = runAsUser

	sc.RunAsUser = t.RunAsUser
	podSpec.SecurityContext = &sc

	return nil
}

func (t *securityContextTrait) getUser(e *Environment) (*int64, error) {
	if t.RunAsUser != nil {
		return t.RunAsUser, nil
	}

	// get security context UID from Openshift when non.configured by the user
	isOpenShift, err := openshift.IsOpenShift(e.Client)
	if err != nil {
		return nil, err
	}
	if !isOpenShift {
		return nil, nil
	}

	runAsUser, err := openshift.GetOpenshiftUser(e.Ctx, e.Client, e.Integration.Namespace)
	if err != nil {
		return nil, err
	}

	return runAsUser, nil
}

func (t *securityContextTrait) getRunAsNonRoot() *bool {
	if t.RunAsNonRoot == nil {
		return ptr.To(defaultPodRunAsNonRoot)
	}

	return t.RunAsNonRoot
}

func (t *securityContextTrait) getSeccompProfileType() corev1.SeccompProfileType {
	if t.SeccompProfileType == "" {
		return defaultPodSeccompProfileType
	}

	return t.SeccompProfileType
}
