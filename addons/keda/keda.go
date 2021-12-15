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

package keda

import (
	"strings"

	kedav1alpha1 "github.com/apache/camel-k/addons/keda/duck/v1alpha1"
	camelv1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	camelv1alpha1 "github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
	scase "github.com/stoewer/go-strcase"
	v1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

// The Keda trait can be used for automatic integration with Keda autoscalers.
//
// The Keda trait is disabled by default.
//
// +camel-k:trait=keda.
type kedaTrait struct {
	trait.BaseTrait `property:",squash"`
	// Enables automatic configuration of the trait.
	Auto *bool `property:"auto" json:"auto,omitempty"`
	// Convert metadata properties to camelCase (needed because trait properties use kebab-case). Enabled by default.
	CamelCaseConversion *bool `property:"camel-case-conversion" json:"camelCaseConversion,omitempty"`
	// Set the spec->replicas field on the top level controller to an explicit value if missing, to allow Keda to recognize it as a scalable resource
	HackControllerReplicas *bool `property:"hack-controller-replicas" json:"hackControllerReplicas,omitempty"`
	// Interval (seconds) to check each trigger on (minimum 10 seconds)
	PollingInterval *int32 `property:"polling-interval" json:"pollingInterval,omitempty"`
	// The wait period between the last active trigger reported and scaling the resource back to 0
	CooldownPeriod *int32 `property:"cooldown-period" json:"cooldownPeriod,omitempty"`
	// Enabling this property allows KEDA to scale the resource down to the specified number of replicas
	IdleReplicaCount *int32 `property:"idle-replica-count" json:"idleReplicaCount,omitempty"`
	// Minimum number of replicas
	MinReplicaCount *int32 `property:"min-replica-count" json:"minReplicaCount,omitempty"`
	// Maximum number of replicas
	MaxReplicaCount *int32 `property:"max-replica-count" json:"maxReplicaCount,omitempty"`
	// Definition of triggers according to the Keda format. Each trigger must contain `type` field corresponding
	// to the name of a Keda autoscaler and a key/value map named `metadata` containing specific trigger options.
	Triggers []kedaTrigger `property:"triggers" json:"triggers,omitempty"`
}

type kedaTrigger struct {
	Type     string            `property:"type" json:"type,omitempty"`
	Metadata map[string]string `property:"metadata" json:"metadata,omitempty"`
}

// NewKedaTrait --.
func NewKedaTrait() trait.Trait {
	return &kedaTrait{
		BaseTrait: trait.NewBaseTrait("keda", trait.TraitOrderPostProcessResources),
	}
}

func (t *kedaTrait) Configure(e *trait.Environment) (bool, error) {
	if t.Enabled == nil || !*t.Enabled {
		return false, nil
	}

	if !e.IntegrationInPhase(camelv1.IntegrationPhaseInitialization) && !e.IntegrationInRunningPhases() {
		return false, nil
	}

	return true, nil
}

func (t *kedaTrait) Apply(e *trait.Environment) error {
	if e.IntegrationInPhase(camelv1.IntegrationPhaseInitialization) {
		if t.HackControllerReplicas == nil || *t.HackControllerReplicas {
			if err := t.hackControllerReplicas(e); err != nil {
				return err
			}
		}
	} else if e.IntegrationInRunningPhases() {
		if so, err := t.getScaledObject(e); err != nil {
			return err
		} else if so != nil {
			e.Resources.Add(so)
		}
	}

	return nil
}

func (t *kedaTrait) getScaledObject(e *trait.Environment) (*kedav1alpha1.ScaledObject, error) {
	if len(t.Triggers) == 0 {
		return nil, nil
	}
	obj := kedav1alpha1.NewScaledObject(e.Integration.Namespace, e.Integration.Name)
	obj.Spec.ScaleTargetRef = t.getTopControllerReference(e)
	if t.PollingInterval != nil {
		obj.Spec.PollingInterval = t.PollingInterval
	}
	if t.CooldownPeriod != nil {
		obj.Spec.CooldownPeriod = t.CooldownPeriod
	}
	if t.IdleReplicaCount != nil {
		obj.Spec.IdleReplicaCount = t.IdleReplicaCount
	}
	if t.MinReplicaCount != nil {
		obj.Spec.MinReplicaCount = t.MinReplicaCount
	}
	if t.MaxReplicaCount != nil {
		obj.Spec.MaxReplicaCount = t.MaxReplicaCount
	}
	for _, trigger := range t.Triggers {
		meta := make(map[string]string)
		for k, v := range trigger.Metadata {
			kk := k
			if t.CamelCaseConversion == nil || *t.CamelCaseConversion {
				kk = scase.LowerCamelCase(k)
			}
			meta[kk] = v
		}
		st := kedav1alpha1.ScaleTriggers{
			Type:     trigger.Type,
			Metadata: meta,
		}
		obj.Spec.Triggers = append(obj.Spec.Triggers, st)
	}

	return &obj, nil
}

func (t *kedaTrait) hackControllerReplicas(e *trait.Environment) error {
	ctrlRef := t.getTopControllerReference(e)
	applier := e.Client.ServerOrClientSideApplier()
	if ctrlRef.Kind == camelv1alpha1.KameletBindingKind {
		// Update the KameletBinding directly (do not add it to env resources, it's the integration parent)
		key := client.ObjectKey{
			Namespace: e.Integration.Namespace,
			Name:      ctrlRef.Name,
		}
		klb := camelv1alpha1.KameletBinding{}
		if err := e.Client.Get(e.Ctx, key, &klb); err != nil {
			return err
		}
		if klb.Spec.Replicas == nil {
			one := int32(1)
			klb.Spec.Replicas = &one
			if err := applier.Apply(e.Ctx, &klb); err != nil {
				return err
			}
		}
	} else {
		if e.Integration.Spec.Replicas == nil {
			one := int32(1)
			e.Integration.Spec.Replicas = &one
			if err := applier.Apply(e.Ctx, e.Integration); err != nil {
				return err
			}
		}
	}
	return nil
}

func (t *kedaTrait) getTopControllerReference(e *trait.Environment) *v1.ObjectReference {
	for _, o := range e.Integration.OwnerReferences {
		if o.Kind == v1alpha1.KameletBindingKind && strings.HasPrefix(o.APIVersion, v1alpha1.SchemeGroupVersion.Group) {
			return &v1.ObjectReference{
				APIVersion: o.APIVersion,
				Kind:       o.Kind,
				Name:       o.Name,
			}
		}
	}
	return &v1.ObjectReference{
		APIVersion: e.Integration.APIVersion,
		Kind:       e.Integration.Kind,
		Name:       e.Integration.Name,
	}
}
