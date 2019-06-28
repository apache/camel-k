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

package integrationplatform

import (
	"context"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"k8s.io/apimachinery/pkg/labels"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"
)

// NewStartAction returns a action that waits for all required platform resources to start
func NewStartAction() Action {
	return &startAction{}
}

type startAction struct {
	baseAction
}

func (action *startAction) Name() string {
	return "start"
}

func (action *startAction) CanHandle(platform *v1alpha1.IntegrationPlatform) bool {
	return platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseStarting || platform.Status.Phase == v1alpha1.IntegrationPlatformPhaseError
}

func (action *startAction) Handle(ctx context.Context, platform *v1alpha1.IntegrationPlatform) (*v1alpha1.IntegrationPlatform, error) {
	aggregatePhase, err := action.aggregatePlatformPhaseFromContexts(ctx, platform.Namespace)
	if err != nil {
		return nil, err
	}

	if platform.Status.Phase != aggregatePhase {
		platform.Status.Phase = aggregatePhase
		return platform, nil
	}

	// wait
	return nil, nil
}

func (action *startAction) aggregatePlatformPhaseFromContexts(ctx context.Context, namespace string) (v1alpha1.IntegrationPlatformPhase, error) {
	ctxs := v1alpha1.NewIntegrationKitList()
	options := k8sclient.ListOptions{
		LabelSelector: labels.SelectorFromSet(labels.Set{
			"camel.apache.org/kit.type": "platform",
		}),
		Namespace: namespace,
	}
	if err := action.client.List(ctx, &options, &ctxs); err != nil {
		return "", err
	}

	countReady := 0
	for _, ctx := range ctxs.Items {
		if ctx.Status.Phase == v1alpha1.IntegrationKitPhaseError {
			return v1alpha1.IntegrationPlatformPhaseError, nil
		} else if ctx.Status.Phase == v1alpha1.IntegrationKitPhaseReady {
			countReady++
		}
	}

	if countReady < len(ctxs.Items) {
		return v1alpha1.IntegrationPlatformPhaseStarting, nil
	}

	return v1alpha1.IntegrationPlatformPhaseReady, nil
}
