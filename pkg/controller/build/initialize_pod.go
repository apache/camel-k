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

package build

import (
	"context"

	"github.com/apache/camel-k/pkg/install"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	k8sclient "sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/apache/camel-k/pkg/apis/camel/v1alpha1"
)

// NewInitializePodAction creates a new initialize action
func NewInitializePodAction() Action {
	return &initializePodAction{}
}

type initializePodAction struct {
	baseAction
}

// Name returns a common name of the action
func (action *initializePodAction) Name() string {
	return "initialize-pod"
}

// CanHandle tells whether this action can handle the build
func (action *initializePodAction) CanHandle(build *v1alpha1.Build) bool {
	return build.Status.Phase == v1alpha1.BuildPhaseInitialization &&
		build.Spec.Platform.Build.BuildStrategy == v1alpha1.IntegrationPlatformBuildStrategyPod
}

// Handle handles the builds
func (action *initializePodAction) Handle(ctx context.Context, build *v1alpha1.Build) (*v1alpha1.Build, error) {
	// Ensure service account is present
	// TODO: maybe this should be done by the platform trait ??
	if err := action.ensureServiceAccount(ctx, build); err != nil {
		return nil, errors.Wrap(err, "cannot ensure service account is present")
	}

	if err := deleteBuilderPod(ctx, action.client, build); err != nil {
		return nil, errors.Wrap(err, "cannot delete build pod")
	}

	pod, err := getBuilderPod(ctx, action.client, build)
	if err != nil || pod != nil {
		// We return and wait for the pod to be deleted before de-queue the build pod.
		return nil, err
	}

	build.Status.Phase = v1alpha1.BuildPhaseScheduling

	return build, nil
}

func (action *initializePodAction) ensureServiceAccount(ctx context.Context, build *v1alpha1.Build) error {
	sa := corev1.ServiceAccount{}
	saKey := k8sclient.ObjectKey{
		Name:      "camel-k-builder",
		Namespace: build.Namespace,
	}

	err := action.client.Get(ctx, saKey, &sa)
	if err != nil && k8serrors.IsNotFound(err) {
		// Create a proper service account
		return install.BuilderServiceAccountRoles(ctx, action.client, build.Namespace)
	}

	return err
}
