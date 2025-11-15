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
	"context"
	"fmt"
	"strings"

	"github.com/apache/camel-k/v2/pkg/util/defaults"

	corev1 "k8s.io/api/core/v1"
	rbacv1 "k8s.io/api/rbac/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	v1 "github.com/apache/camel-k/v2/pkg/apis/camel/v1"
	"github.com/apache/camel-k/v2/pkg/platform"
	"github.com/apache/camel-k/v2/pkg/trait"
	"github.com/apache/camel-k/v2/pkg/util/kubernetes"
)

// NewBuildAction creates a new build request handling action for the kit.
func NewBuildAction() Action {
	return &buildAction{}
}

type buildAction struct {
	baseAction
}

func (action *buildAction) Name() string {
	return "build"
}

func (action *buildAction) CanHandle(kit *v1.Integration) bool {
	return kit.Status.Phase == v1.IntegrationPhaseBuildSubmitted || kit.Status.Phase == v1.IntegrationPhaseBuildRunning
}

func (action *buildAction) Handle(ctx context.Context, it *v1.Integration) (*v1.Integration, error) {
	switch it.Status.Phase {
	case v1.IntegrationPhaseBuildSubmitted:
		return action.handleBuildSubmitted(ctx, it)
	case v1.IntegrationPhaseBuildRunning:
		return action.handleBuildRunning(ctx, it)
	}

	return nil, nil
}

func (action *buildAction) handleBuildSubmitted(ctx context.Context, it *v1.Integration) (*v1.Integration, error) {
	build, err := kubernetes.GetBuild(ctx, action.client, it.Name, it.Namespace)
	if err != nil && !k8serrors.IsNotFound(err) {
		return nil, err
	}

	// If build is running we need to wait for it to complete or we need to schedule it for interruption
	if build != nil && build.Status.Phase == v1.BuildPhaseRunning {
		action.L.Infof("build %s is still running. Integration %s needs to wait for it to complete", build.Name, it.Name)

		return nil, nil
	}

	if err := action.createBuild(ctx, it); err != nil {
		return nil, err
	}

	// We assume the previously initialized build is running. Future reconciliation
	// will take care of any build status drift.
	it.Status.Phase = v1.IntegrationPhaseBuildRunning

	return it, nil
}

func (action *buildAction) createBuild(ctx context.Context, it *v1.Integration) error {
	env, err := trait.Apply(ctx, action.client, it, nil)
	if err != nil {
		return err
	}

	labels := kubernetes.FilterCamelCreatorLabels(it.Labels)
	annotations := make(map[string]string)

	operatorID := defaults.OperatorID()
	if operatorID != "" {
		annotations[v1.OperatorIDAnnotation] = operatorID
	}

	timeout := env.Platform.Status.Build.GetTimeout()

	// We may need to change certain builder configuration values
	buildConfig := v1.ConfigurationTasksByName(env.Pipeline, "builder")
	if buildConfig.IsEmpty() {
		// default to IntegrationPlatform configuration
		buildConfig = &env.Platform.Status.Build.BuildConfiguration
	} else {
		if buildConfig.Strategy == "" {
			// we always need to define a strategy, so we default to platform if none
			buildConfig.Strategy = env.Platform.Status.Build.BuildConfiguration.Strategy
		}

		if buildConfig.OrderStrategy == "" {
			// we always need to define an order strategy, so we default to platform if none
			buildConfig.OrderStrategy = env.Platform.Status.Build.BuildConfiguration.OrderStrategy
		}
	}

	// The build operation, when executed as a Pod, should be executed by a container image containing the
	// `kamel builder` command. Likely the same image running the operator should be fine.
	buildConfig.ToolImage = platform.OperatorImage
	buildConfig.BuilderPodNamespace = it.Namespace
	v1.SetBuilderConfigurationTasks(env.Pipeline, buildConfig)

	// We need to ensure the presence of the camel-k-builder service account
	// and the related privileges needed to run the builder Pod.
	if err := action.applyBuilderRBAC(ctx, it.Namespace); err != nil {
		return err
	}

	build := &v1.Build{
		TypeMeta: metav1.TypeMeta{
			APIVersion: v1.SchemeGroupVersion.String(),
			Kind:       v1.BuildKind,
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace:   it.Namespace,
			Name:        it.Name,
			Labels:      labels,
			Annotations: annotations,
		},
		Spec: v1.BuildSpec{
			Tasks:   env.Pipeline,
			Timeout: timeout,
		},
	}

	// Set the integration instance as the owner and controller
	if err := controllerutil.SetControllerReference(it, build, action.client.GetScheme()); err != nil {
		return err
	}

	err = action.client.Delete(ctx, build)
	if err != nil && !k8serrors.IsNotFound(err) {
		return fmt.Errorf("cannot delete build: %w", err)
	}

	err = action.client.Create(ctx, build)
	if err != nil {
		return fmt.Errorf("cannot create build: %w", err)
	}

	return nil
}

// applyBuilderRBAC is in charge to verify the existence of the required ServiceAccount
// Role and RoleBinding and create them if they don't exist in a given namespace.
func (action *buildAction) applyBuilderRBAC(ctx context.Context, ns string) error {
	sa, err := kubernetes.LookupServiceAccount(ctx, action.client, ns, platform.BuilderServiceAccount)
	if err != nil {
		return err
	}
	if sa == nil {
		if err := action.createBuilderServiceAccount(ctx, ns); err != nil {
			return err
		}
	}
	r, err := kubernetes.LookupRole(ctx, action.client, ns, platform.BuilderServiceAccount)
	if err != nil {
		return err
	}
	if r == nil {
		if err := action.createBuilderRole(ctx, ns); err != nil {
			return err
		}
	}
	rb, err := kubernetes.LookupRoleBinding(ctx, action.client, ns, platform.BuilderServiceAccount)
	if err != nil {
		return err
	}
	if rb == nil {
		if err := action.createBuilderRoleBinding(ctx, ns); err != nil {
			return err
		}
	}

	return nil
}

func (action *buildAction) handleBuildRunning(ctx context.Context, it *v1.Integration) (*v1.Integration, error) {
	build, err := kubernetes.GetBuild(ctx, action.client, it.Name, it.Namespace)
	if err != nil {
		return nil, err
	}

	switch build.Status.Phase {
	case v1.BuildPhaseRunning:
		action.L.Debug("Build still running")
	case v1.BuildPhaseSucceeded:
		it.Status.Image = build.Status.Image
		artifact, err := build.Status.GetRunnable()
		if err != nil {
			return nil, fmt.Errorf("could not get a runnable artifact from Build due to %w", err)
		}
		it.Status.Jar = artifact.Target
		// Address the image by repository digest instead of tag if possible
		if build.Status.Digest != "" {
			image := it.Status.Image
			i := strings.LastIndex(image, ":")
			if i > 0 {
				image = image[:i]
			}
			it.Status.Image = fmt.Sprintf("%s@%s", image, build.Status.Digest)
		}
		if it.Annotations[v1.IntegrationDontRunAfterBuildAnnotation] == v1.IntegrationDontRunAfterBuildAnnotationTrueValue {
			it.Status.Phase = v1.IntegrationPhaseBuildComplete
		} else {
			it.Status.Phase = v1.IntegrationPhaseDeploying
		}
	case v1.BuildPhaseError, v1.BuildPhaseInterrupted, v1.BuildPhaseFailed:
		it.Status.Phase = v1.IntegrationPhaseError
		reason := fmt.Sprintf("Build%s", build.Status.Phase)
		message := ""
		if build.Status.Failure != nil {
			message = build.Status.Failure.Reason
		}
		it.Status.SetCondition(v1.IntegrationConditionReady, corev1.ConditionFalse, reason, message)
	}

	return it, nil
}

// createBuilderServiceAccount creates the builder SA in the ns namespace.
func (action *buildAction) createBuilderServiceAccount(ctx context.Context, ns string) error {
	sa := &corev1.ServiceAccount{
		TypeMeta: metav1.TypeMeta{
			Kind:       "ServiceAccount",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      platform.BuilderServiceAccount,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
	}
	action.L.Info("Creating %s ServiceAccount in namespace %s", sa.Name, sa.Namespace)

	return action.client.Create(ctx, sa)
}

// createBuilderRole creates the builder role in the ns namespace.
func (action *buildAction) createBuilderRole(ctx context.Context, ns string) error {
	r := &rbacv1.Role{
		TypeMeta: metav1.TypeMeta{
			Kind:       "Role",
			APIVersion: corev1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      platform.BuilderServiceAccount,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Rules: []rbacv1.PolicyRule{
			{
				APIGroups: []string{"camel.apache.org"},
				Resources: []string{"builds"},
				Verbs:     []string{"get", "list"},
			},
			{
				APIGroups: []string{"camel.apache.org"},
				Resources: []string{"builds/status"},
				Verbs:     []string{"get", "patch", "update"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"secrets"},
				Verbs:     []string{"get"},
			},
			{
				APIGroups: []string{""},
				Resources: []string{"configmaps"},
				Verbs:     []string{"get", "list"},
			},
		},
	}
	action.L.Info("Creating %s Role in namespace %s", r.Name, r.Namespace)

	return action.client.Create(ctx, r)
}

// createBuilderRole creates the builder role in the ns namespace.
func (action *buildAction) createBuilderRoleBinding(ctx context.Context, ns string) error {
	rb := &rbacv1.RoleBinding{
		TypeMeta: metav1.TypeMeta{
			Kind:       "RoleBinding",
			APIVersion: rbacv1.SchemeGroupVersion.String(),
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: ns,
			Name:      platform.BuilderServiceAccount,
			Labels: map[string]string{
				"app": "camel-k",
			},
		},
		Subjects: []rbacv1.Subject{
			{
				Kind:      "ServiceAccount",
				Name:      platform.BuilderServiceAccount,
				Namespace: ns,
			},
		},
		// We assume this ClusterRole exists as part as a regular operator installation.
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "Role",
			Name:     platform.BuilderServiceAccount,
		},
	}
	action.L.Info("Creating %s RoleBinding in namespace %s", rb.Name, rb.Namespace)

	return action.client.Create(ctx, rb)
}
