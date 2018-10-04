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

package integration

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/trait"
	"github.com/apache/camel-k/pkg/util/kubernetes"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
)

// NewDeployAction create an action that handles integration deploy
func NewDeployAction() Action {
	return &deployAction{}
}

type deployAction struct {
}

func (action *deployAction) Name() string {
	return "deploy"
}

func (action *deployAction) CanHandle(integration *v1alpha1.Integration) bool {
	return integration.Status.Phase == v1alpha1.IntegrationPhaseDeploying
}

func (action *deployAction) Handle(integration *v1alpha1.Integration) error {
	environment, err := trait.NewEnvironment(integration)
	if err != nil {
		return err
	}
	resources := kubernetes.NewCollection()
	customizers := trait.CustomizersFor(*environment)
	// invoke the trait framework to determine the needed resources
	if _, err = customizers.Customize(*environment, resources); err != nil {
		return errors.Wrap(err, "error during trait customization")
	}
	// TODO we should look for objects that are no longer present in the collection and remove them
	err = action.createOrUpdateObjects(resources.Items(), integration)
	if err != nil {
		return err
	}

	target := integration.DeepCopy()
	logrus.Info("Integration ", target.Name, " transitioning to state ", v1alpha1.IntegrationPhaseRunning)
	target.Status.Phase = v1alpha1.IntegrationPhaseRunning

	return sdk.Update(target)
}

func (action *deployAction) createOrUpdateObjects(objects []runtime.Object, integration *v1alpha1.Integration) error {
	for _, object := range objects {
		err := sdk.Create(object)
		if err != nil && k8serrors.IsAlreadyExists(err) {
			err = sdk.Update(object)
		}
		if err != nil {
			return errors.Wrap(err, "could not create or replace resource for integration "+integration.Name)
		}
	}

	return nil
}
