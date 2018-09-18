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

package action

import (
	"context"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"github.com/operator-framework/operator-sdk/pkg/sdk"
	"github.com/sirupsen/logrus"

	"github.com/apache/camel-k/pkg/build/api"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
	"github.com/apache/camel-k/pkg/build"
)

// NewIntegrationContextBuildAction creates a new build handling action for the context
func NewIntegrationContextBuildAction(ctx context.Context, namespace string) IntegrationContextAction {
	return &integrationContextBuildAction{
		buildManager: build.NewManager(ctx, namespace),
	}
}

type integrationContextBuildAction struct {
	buildManager *build.Manager
}

func (action *integrationContextBuildAction) Name() string {
	return "build"
}

func (action *integrationContextBuildAction) CanHandle(context *v1alpha1.IntegrationContext) bool {
	return context.Status.Phase == v1alpha1.IntegrationContextPhaseBuilding
}

func (action *integrationContextBuildAction) Handle(context *v1alpha1.IntegrationContext) error {
	buildIdentifier := api.BuildIdentifier{
		Name:      "context-" + context.Name,
		Qualifier: context.ResourceVersion,
	}

	buildResult := action.buildManager.Get(buildIdentifier)
	if buildResult.Status == api.BuildStatusNotRequested {
		action.buildManager.Start(api.BuildSource{
			Identifier:   buildIdentifier,
			Dependencies: context.Spec.Dependencies,
		})
		logrus.Info("Build started")
	} else if buildResult.Status == api.BuildStatusError {
		target := context.DeepCopy()
		target.Status.Phase = v1alpha1.IntegrationContextPhaseError
		return sdk.Update(target)
	} else if buildResult.Status == api.BuildStatusCompleted {
		target := context.DeepCopy()
		target.Status.Image = buildResult.Image
		target.Status.Phase = v1alpha1.IntegrationContextPhaseReady
		if err := sdk.Update(target); err != nil {
			return err
		}
		if err := action.informIntegrations(target); err != nil {
			return err
		}
	}

	return nil
}

// informIntegrations triggers the processing of all integrations waiting for this context to be built
func (action *integrationContextBuildAction) informIntegrations(context *v1alpha1.IntegrationContext) error {
	list := v1alpha1.NewIntegrationList()
	err := sdk.List(context.Namespace, &list, sdk.WithListOptions(&metav1.ListOptions{}))
	if err != nil {
		return err
	}
	for _, integration := range list.Items {
		if integration.Spec.Context != context.Name {
			continue
		}

		if integration.Annotations == nil {
			integration.Annotations = make(map[string]string)
		}
		integration.Annotations["camel.apache.org/context.digest"] = context.Status.Digest
		err = sdk.Update(&integration)
		if err != nil {
			return err
		}
	}
	return nil
}
