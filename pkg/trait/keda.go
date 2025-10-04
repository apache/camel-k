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
	"strings"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	"github.com/apache/camel-k/v2/pkg/apis/duck/keda/v1alpha1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/ptr"
)

const (
	kedaTraitID = "keda"
)

type kedaTrait struct {
	BaseTrait
	traitv1.KedaTrait `property:",squash"`
}

func newKedaTrait() Trait {
	return &kedaTrait{
		BaseTrait: NewBaseTrait(kedaTraitID, TraitOrderPostProcessResources),
	}
}

func (t *kedaTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	if e.Integration == nil || !ptr.Deref(t.Enabled, false) || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	return len(t.Triggers) > 0, nil, nil
}

func (t *kedaTrait) Apply(e *Environment) error {
	triggers, auths := t.populateTriggers(e.Integration.Name, e.Integration.Namespace)
	scaleTarget := t.getScaleTarget(e.Integration)
	scaledObject := &v1alpha1.ScaledObject{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "ScaledObject",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      e.Integration.Name,
			Namespace: e.Integration.Namespace,
		},
		Spec: v1alpha1.ScaledObjectSpec{
			ScaleTargetRef:   scaleTarget,
			PollingInterval:  t.PollingInterval,
			CooldownPeriod:   t.CooldownPeriod,
			IdleReplicaCount: t.IdleReplicaCount,
			MinReplicaCount:  t.MinReplicaCount,
			MaxReplicaCount:  t.MaxReplicaCount,
			Triggers:         triggers,
		},
	}
	for _, auth := range auths {
		e.Resources.Add(auth)
	}
	e.Resources.Add(scaledObject)
	return nil
}

func (t *kedaTrait) populateTriggers(itName, itNamespace string) ([]v1alpha1.ScaleTriggers, []*v1alpha1.TriggerAuthentication) {
	var auths []*v1alpha1.TriggerAuthentication
	triggers := make([]v1alpha1.ScaleTriggers, 0, len(t.Triggers))
	for _, trigger := range t.Triggers {
		scaleTrigger := v1alpha1.ScaleTriggers{
			Type:     trigger.Type,
			Metadata: trigger.Metadata,
		}
		if trigger.Secrets != nil {
			triggerAuth := populateTriggerAuth(trigger.Secrets, itName, itNamespace, trigger.Type)
			auths = append(auths, triggerAuth)
			scaleTrigger.AuthenticationRef = &v1alpha1.ScaledObjectAuthRef{
				Name: triggerAuth.Name,
			}
		}
		triggers = append(triggers, scaleTrigger)
	}

	return triggers, auths
}

func populateTriggerAuth(secrets []*traitv1.KedaSecret, itName, itNamespace, kedaType string) *v1alpha1.TriggerAuthentication {
	triggerAuth := &v1alpha1.TriggerAuthentication{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1alpha1.SchemeGroupVersion.String(),
			Kind:       "TriggerAuthentication",
		},
		ObjectMeta: metav1.ObjectMeta{
			Name:      fmt.Sprintf("%s-%s", itName, kedaType),
			Namespace: itNamespace,
		},
		Spec: v1alpha1.TriggerAuthenticationSpec{
			SecretTargetRef: make([]v1alpha1.AuthSecretTargetRef, 0, len(secrets)),
		},
	}
	for _, secret := range secrets {
		for k, v := range secret.Mapping {
			authSecretTargetRef := v1alpha1.AuthSecretTargetRef{Name: secret.Name}
			authSecretTargetRef.Key = k
			authSecretTargetRef.Parameter = v

			triggerAuth.Spec.SecretTargetRef = append(triggerAuth.Spec.SecretTargetRef, authSecretTargetRef)
		}
	}

	return triggerAuth
}

// getScaleTarget returns either an Integration or a Pipe, if the Integration was created by a Pipe.
func (t *kedaTrait) getScaleTarget(it *v1.Integration) *corev1.ObjectReference {
	for _, o := range it.OwnerReferences {
		if o.Kind == v1.PipeKind && strings.HasPrefix(o.APIVersion, v1.SchemeGroupVersion.Group) {
			return &corev1.ObjectReference{
				APIVersion: o.APIVersion,
				Kind:       o.Kind,
				Name:       o.Name,
			}
		}
	}
	return &corev1.ObjectReference{
		APIVersion: it.APIVersion,
		Kind:       it.Kind,
		Name:       it.Name,
	}
}
