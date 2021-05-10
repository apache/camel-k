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

	"github.com/pkg/errors"

	corev1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	ctrl "sigs.k8s.io/controller-runtime/pkg/client"

	v1 "github.com/apache/camel-k/pkg/apis/camel/v1"
)

func newInitializePodAction() Action {
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
func (action *initializePodAction) CanHandle(build *v1.Build) bool {
	return build.Status.Phase == v1.BuildPhaseInitialization
}

// Handle handles the builds
func (action *initializePodAction) Handle(ctx context.Context, build *v1.Build) (*v1.Build, error) {
	if err := deleteBuilderPod(ctx, action.client, build); err != nil {
		return nil, errors.Wrap(err, "cannot delete build pod")
	}

	pod, err := getBuilderPod(ctx, action.client, build)
	if err != nil || pod != nil {
		// We return and wait for the pod to be deleted before de-queue the build pod.
		return nil, err
	}

	build.Status.Phase = v1.BuildPhaseScheduling

	return build, nil
}

func deleteBuilderPod(ctx context.Context, client ctrl.Writer, build *v1.Build) error {
	pod := corev1.Pod{
		TypeMeta: metav1.TypeMeta{
			APIVersion: corev1.SchemeGroupVersion.String(),
			Kind:       "Pod",
		},
		ObjectMeta: metav1.ObjectMeta{
			Namespace: build.Namespace,
			Name:      buildPodName(build),
		},
	}

	err := client.Delete(ctx, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil
	}

	return err
}

func getBuilderPod(ctx context.Context, client ctrl.Reader, build *v1.Build) (*corev1.Pod, error) {
	pod := corev1.Pod{}
	err := client.Get(ctx, ctrl.ObjectKey{Namespace: build.Namespace, Name: buildPodName(build)}, &pod)
	if err != nil && k8serrors.IsNotFound(err) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	return &pod, nil
}

func buildPodName(build *v1.Build) string {
	return "camel-k-" + build.Name + "-builder"
}
