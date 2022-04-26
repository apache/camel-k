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
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/utils/pointer"

	serving "knative.dev/serving/pkg/apis/serving/v1"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
	traitv1 "github.com/apache/camel-k/pkg/apis/camel/v1/trait"
)

type ownerTrait struct {
	BaseTrait
	traitv1.OwnerTrait `property:",squash"`
}

func newOwnerTrait() Trait {
	return &ownerTrait{
		BaseTrait: NewBaseTrait("owner", 2500),
	}
}

func (t *ownerTrait) Configure(e *Environment) (bool, error) {
	if !pointer.BoolDeref(t.Enabled, true) {
		return false, nil
	}

	if e.Integration == nil {
		return false, nil
	}

	return e.IntegrationInPhase(v1.IntegrationPhaseInitialization) || e.IntegrationInRunningPhases(), nil
}

func (t *ownerTrait) Apply(e *Environment) error {
	controller := true
	blockOwnerDeletion := true

	targetLabels := make(map[string]string)
	if e.Integration.Labels != nil {
		for _, k := range t.TargetLabels {
			if v, ok := e.Integration.Labels[k]; ok {
				targetLabels[k] = v
			}
		}
	}

	targetAnnotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for _, k := range t.TargetAnnotations {
			if v, ok := e.Integration.Annotations[k]; ok {
				targetAnnotations[k] = v
			}
		}
	}

	e.Resources.VisitMetaObject(func(res metav1.Object) {
		// Cross-namespace references are forbidden and also asynchronously refused
		// by the api server (sometimes no error is thrown but the resource is not created).
		// Ref: https://github.com/kubernetes/kubernetes/issues/65200
		if res.GetNamespace() == "" || res.GetNamespace() == e.Integration.Namespace {
			references := []metav1.OwnerReference{
				{
					APIVersion:         e.Integration.APIVersion,
					Kind:               e.Integration.Kind,
					Name:               e.Integration.Name,
					UID:                e.Integration.UID,
					Controller:         &controller,
					BlockOwnerDeletion: &blockOwnerDeletion,
				},
			}
			res.SetOwnerReferences(references)
		}

		// Transfer annotations
		t.propagateLabelAndAnnotations(res, targetLabels, targetAnnotations)
	})

	e.Resources.VisitDeployment(func(deployment *appsv1.Deployment) {
		t.propagateLabelAndAnnotations(&deployment.Spec.Template, targetLabels, targetAnnotations)
	})

	e.Resources.VisitKnativeService(func(service *serving.Service) {
		t.propagateLabelAndAnnotations(&service.Spec.ConfigurationSpec.Template, targetLabels, targetAnnotations)
	})

	return nil
}

// IsPlatformTrait overrides base class method.
func (t *ownerTrait) IsPlatformTrait() bool {
	return true
}

func (t *ownerTrait) propagateLabelAndAnnotations(res metav1.Object, targetLabels map[string]string, targetAnnotations map[string]string) {
	// Transfer annotations
	annotations := res.GetAnnotations()
	if annotations == nil {
		annotations = make(map[string]string)
	}

	for k, v := range targetAnnotations {
		if _, ok := annotations[k]; !ok {
			annotations[k] = v
		}
	}
	res.SetAnnotations(annotations)

	// Transfer labels
	labels := res.GetLabels()
	if labels == nil {
		labels = make(map[string]string)
	}

	for k, v := range targetLabels {
		if _, ok := labels[k]; !ok {
			labels[k] = v
		}
	}
	res.SetLabels(labels)
}
