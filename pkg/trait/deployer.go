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
	"errors"
	"fmt"

	traitv1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1/trait"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime/pkg/client"
)

const (
	deployerTraitID    = "deployer"
	deployerTraitOrder = 900
)

type deployerTrait struct {
	BasePlatformTrait
	traitv1.DeployerTrait `property:",squash"`
}

var _ ControllerStrategySelector = &deployerTrait{}

func newDeployerTrait() Trait {
	return &deployerTrait{
		BasePlatformTrait: NewBasePlatformTrait(deployerTraitID, deployerTraitOrder),
	}
}

func (t *deployerTrait) Configure(e *Environment) (bool, *TraitCondition, error) {
	var condition *TraitCondition

	return e.Integration != nil, condition, nil
}

func (t *deployerTrait) Apply(e *Environment) error {
	e.PostActions = append(e.PostActions, func(env *Environment) error {
		applier := e.Client.ServerOrClientSideApplier()
		snapshots := make([]resourceSnapshot, 0, env.Resources.Size())

		for _, resource := range env.Resources.Items() {
			previous := resource.DeepCopyObject().(ctrl.Object)
			err := e.Client.Get(e.Ctx, ctrl.ObjectKeyFromObject(resource), previous)

			switch {
			case k8serrors.IsNotFound(err):
				previous = nil
			case err != nil:
				operationErr := fmt.Errorf(
					"unable to snapshot resource %s/%s: %w",
					resource.GetNamespace(), resource.GetName(), err,
				)
				rollbackErr := rollbackResources(e, snapshots)
				return fatalDeploymentError(operationErr, rollbackErr)
			}

			// Include the current resource before applying it because an apply
			// can fail after the API server has persisted the changes.
			snapshots = append(snapshots, resourceSnapshot{
				resource: resource,
				previous: previous,
			})

			if err := applier.Apply(e.Ctx, resource); err != nil {
				operationErr := fmt.Errorf(
					"unable to apply resource %s/%s: %w",
					resource.GetNamespace(), resource.GetName(), err,
				)
				rollbackErr := rollbackResources(e, snapshots)
				return fatalDeploymentError(operationErr, rollbackErr)
			}
		}

		return nil
	})

	return nil
}

type resourceSnapshot struct {
	resource ctrl.Object
	previous ctrl.Object
}

func rollbackResources(e *Environment, snapshots []resourceSnapshot) error {
	var rollbackErrors []error

	for i := len(snapshots) - 1; i >= 0; i-- {
		snapshot := snapshots[i]

		if snapshot.previous == nil {
			current := snapshot.resource.DeepCopyObject().(ctrl.Object)
			err := e.Client.Get(e.Ctx, ctrl.ObjectKeyFromObject(snapshot.resource), current)

			if k8serrors.IsNotFound(err) {
				continue
			}
			if err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf(
					"unable to get resource %s/%s during rollback: %w",
					snapshot.resource.GetNamespace(), snapshot.resource.GetName(), err,
				))
				continue
			}

			if err := e.Client.Delete(e.Ctx, current); err != nil && !k8serrors.IsNotFound(err) {
				rollbackErrors = append(rollbackErrors, fmt.Errorf(
					"unable to delete newly created resource %s/%s during rollback: %w",
					snapshot.resource.GetNamespace(), snapshot.resource.GetName(), err,
				))
			}
			continue
		}

		current := snapshot.previous.DeepCopyObject().(ctrl.Object)
		err := e.Client.Get(e.Ctx, ctrl.ObjectKeyFromObject(snapshot.previous), current)

		if k8serrors.IsNotFound(err) {
			restored := snapshot.previous.DeepCopyObject().(ctrl.Object)
			restored.SetResourceVersion("")
			restored.SetUID("")
			restored.SetCreationTimestamp(metav1.Time{})
			restored.SetManagedFields(nil)

			if createErr := e.Client.Create(e.Ctx, restored); createErr != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf(
					"unable to recreate resource %s/%s during rollback: %w",
					restored.GetNamespace(), restored.GetName(), createErr,
				))
			}
			continue
		}

		if err != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf(
				"unable to get resource %s/%s during rollback: %w",
				snapshot.previous.GetNamespace(), snapshot.previous.GetName(), err,
			))
			continue
		}

		restored := snapshot.previous.DeepCopyObject().(ctrl.Object)
		restored.SetResourceVersion(current.GetResourceVersion())
		restored.SetManagedFields(current.GetManagedFields())

		if updateErr := e.Client.Update(e.Ctx, restored); updateErr != nil {
			rollbackErrors = append(rollbackErrors, fmt.Errorf(
				"unable to restore resource %s/%s during rollback: %w",
				restored.GetNamespace(), restored.GetName(), updateErr,
			))
		}
	}

	return errors.Join(rollbackErrors...)
}

func fatalDeploymentError(operationErr, rollbackErr error) error {
	if rollbackErr != nil {
		return newFatalPostActionError(fmt.Errorf(
			"integration upgrade did not go through; rollback was incomplete: %w",
			errors.Join(operationErr, rollbackErr),
		))
	}

	return newFatalPostActionError(fmt.Errorf(
		"integration upgrade did not go through; previous generation has been preserved: %w",
		operationErr,
	))
}

func (t *deployerTrait) SelectControllerStrategy(e *Environment) (*ControllerStrategy, error) {
	//nolint:staticcheck
	if t.Kind != "" {
		strategy := ControllerStrategy(t.Kind)

		return &strategy, nil
	}

	return nil, nil
}

func (t *deployerTrait) ControllerStrategySelectorOrder() int {
	return 0
}

// RequiresIntegrationPlatform overrides base class method.
func (t *deployerTrait) RequiresIntegrationPlatform() bool {
	return false
}
