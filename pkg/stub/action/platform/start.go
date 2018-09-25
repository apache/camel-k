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
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
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
	aggregatePhase, err := action.aggregatePlatformPhaseFromContexts(platform.Namespace)
	if err != nil {
		return err
	}
	if platform.Status.Phase != aggregatePhase {
		target := platform.DeepCopy()
		logrus.Info("Platform ", target.Name, " transitioning to state ", aggregatePhase)
		target.Status.Phase = aggregatePhase
		return sdk.Update(target)
	}
	// wait
	return nil
}

func (action *startAction) aggregatePlatformPhaseFromContexts(namespace string) (v1alpha1.IntegrationPlatformPhase, error) {
	ctxs := v1alpha1.NewIntegrationContextList()
	options := metav1.ListOptions{
		LabelSelector: "camel.apache.org/context.type=platform",
	}
	if err := sdk.List(namespace, &ctxs, sdk.WithListOptions(&options)); err != nil {
		return "", err
	}

	countReady := 0
	for _, ctx := range ctxs.Items {
		if ctx.Status.Phase == v1alpha1.IntegrationContextPhaseError {
			return v1alpha1.IntegrationPlatformPhaseError, nil
		} else if ctx.Status.Phase == v1alpha1.IntegrationContextPhaseReady {
			countReady++
		}
	}

	if countReady < len(ctxs.Items) {
		return v1alpha1.IntegrationPlatformPhaseStarting, nil
	}

	return v1alpha1.IntegrationPlatformPhaseReady, nil
}
