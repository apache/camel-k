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
	corev1 "k8s.io/api/core/v1"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
)

const (
	pullSecretTraitID    = "pull-secret"
	pullSecretTraitOrder = 1700
)

type pullSecretTrait struct {
	BaseTrait
	traitv1.PullSecretTrait `property:",squash"`
}

func newPullSecretTrait() Trait {
	return &pullSecretTrait{
		BaseTrait: NewBaseTrait(pullSecretTraitID, pullSecretTraitOrder),
	}
}

func (t *pullSecretTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	var condition *TraitCondition
	if e.Integration == nil || !e.IntegrationInRunningPhases() {
		return false, nil, nil
	}

	return t.SecretName != "", condition, nil
}

func (t *pullSecretTrait) Apply(e *Environment) error {
	if t.SecretName != "" {
		e.Resources.VisitPodSpec(func(p *corev1.PodSpec) {
			p.ImagePullSecrets = append(p.ImagePullSecrets, corev1.LocalObjectReference{
				Name: t.SecretName,
			})
		})
	}

	return nil
}
