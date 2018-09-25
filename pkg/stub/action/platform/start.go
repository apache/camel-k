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

package platform

import (
	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"
)

// NewStartAction returns a action that waits for all required platform resources to start
func NewStartAction() Action {
	return &startAction{}
}

type startAction struct {
}

func (action *startAction) Name() string {
	return "start"
}

func (action *startAction) CanHandle(platform *v1alpha1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseStarting || platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseError
}

func (action *startAction) Handle(platform *v1alpha1.IntegrationPlatform) error {

	coreStatus, err := action.getContextReady(platform.Namespace, "core")
	if err != nil {
		return err
	}

	groovyStatus, err := action.getContextReady(platform.Namespace, "groovy")
	if err != nil {
		return err
	}

	if coreStatus == v1alpha1.IntegrationContextPhaseError || groovyStatus == v1alpha1.IntegrationContextPhaseError {
		if platform.Status.Phase != v1alpha1.IntegrationPlatformPhaseError {
			target := platform.DeepCopy()
			logrus.Info("Platform ", target.Name, " transitioning to state ", v1alpha1.IntegrationPlatformPhaseError)
			target.Status.Phase = v1alpha1.IntegrationPlatformPhaseError
			return sdk.Update(target)
		}
		return nil
	} else if coreStatus == v1alpha1.IntegrationContextPhaseReady && groovyStatus == v1alpha1.IntegrationContextPhaseReady {
		target := platform.DeepCopy()
		logrus.Info("Platform ", target.Name, " transitioning to state ", v1alpha1.IntegrationPlatformPhaseReady)
		target.Status.Phase = v1alpha1.IntegrationPlatformPhaseReady
		return sdk.Update(target)
	}

	// wait
	return nil
}

func (action *startAction) getContextReady(namespace string, name string) (v1alpha1.IntegrationContextPhase, error) {
	ctx := v1alpha1.NewIntegrationContext(namespace, name)
	if err := sdk.Get(&ctx); err != nil {
		return "", err
	}
	return ctx.Status.Phase, nil
}
