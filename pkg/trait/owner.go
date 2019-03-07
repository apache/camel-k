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
	"strings"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/util/finalizer"

	"github.com/pkg/errors"

	serving "github.com/knative/serving/pkg/apis/serving/v1alpha1"
	appsv1 "k8s.io/api/apps/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

// ownerTrait ensures that all created resources belong to the integration being created
// and transfers annotations and labels on the integration onto these owned resources being created
type ownerTrait struct {
	BaseTrait `property:",squash"`

	TargetAnnotations string `property:"target-annotations"`
	TargetLabels      string `property:"target-labels"`
}

func newOwnerTrait() *ownerTrait {
	return &ownerTrait{
		BaseTrait: newBaseTrait("owner"),
	}
}

func (t *ownerTrait) Configure(e *Environment) (bool, error) {
	if t.Enabled != nil && !*t.Enabled {
		return false, nil
	}

	if e.Integration == nil {
		return false, nil
	}

	return e.IntegrationInPhase(v1alpha1.IntegrationPhaseDeploying), nil
}

func (t *ownerTrait) Apply(e *Environment) error {
	controller := true
	blockOwnerDeletion := true

	targetLabels := make(map[string]string)
	if e.Integration.Labels != nil {
		for _, k := range strings.Split(t.TargetLabels, ",") {
			if v, ok := e.Integration.Labels[k]; ok {
				targetLabels[k] = v
			}
		}
	}

	targetAnnotations := make(map[string]string)
	if e.Integration.Annotations != nil {
		for _, k := range strings.Split(t.TargetAnnotations, ",") {
			if v, ok := e.Integration.Annotations[k]; ok {
				targetAnnotations[k] = v
			}
		}
	}

	ok, err := finalizer.Exists(e.Integration, finalizer.CamelIntegrationFinalizer)
	if err != nil {
		return errors.Wrap(err, "failed to read finalizer"+finalizer.CamelIntegrationFinalizer)
	}

	e.Resources.VisitMetaObject(func(res metav1.Object) {
		//
		// do not add owner reference if the finalizer is set
		// so resources are not automatically deleted by k8s
		// when owner is deleted
		//
		if !ok {
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
		t.propagateLabelAndAnnotations(&service.Spec.RunLatest.Configuration.RevisionTemplate, targetLabels, targetAnnotations)
	})

	return nil
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
